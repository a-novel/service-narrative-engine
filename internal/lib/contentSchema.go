package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"unicode/utf8"

	"github.com/google/jsonschema-go/jsonschema"
)

var (
	// ErrContentSchemaInvalid reports a schema that cannot be prepared for validation.
	ErrContentSchemaInvalid = errors.New("invalid content schema")

	errContentDocumentEmpty         = errors.New("json document is empty")
	errContentDocumentInvalid       = errors.New("json document is invalid")
	errContentDocumentNotObject     = errors.New("json value must be an object")
	errContentDocumentSchemaInvalid = errors.New("json document does not match its schema")
	errContentDocumentTooLarge      = errors.New("json document exceeds the size limit")
)

// ContentSchema validates complete or partial object documents against one JSON Schema.
// Each form is prepared once, on its first use.
type ContentSchema struct {
	outputSchema    json.RawMessage
	maxBytes        int
	resolveComplete func() (*jsonschema.Resolved, error)
	resolvePartial  func() (*jsonschema.Resolved, error)
}

// NewContentSchema creates a lazy validator. A non-positive maxBytes disables
// the document-size limit.
func NewContentSchema(outputSchema json.RawMessage, maxBytes int) *ContentSchema {
	outputSchema = bytes.Clone(outputSchema)

	resolveComplete := sync.OnceValues(func() (*jsonschema.Resolved, error) {
		return resolveContentSchema(outputSchema)
	})
	resolvePartial := sync.OnceValues(func() (*jsonschema.Resolved, error) {
		partialSchema, err := buildPartialContentSchema(outputSchema)
		if err != nil {
			return nil, err
		}

		return resolveContentSchema(partialSchema)
	})

	return &ContentSchema{
		outputSchema:    outputSchema,
		maxBytes:        maxBytes,
		resolveComplete: resolveComplete,
		resolvePartial:  resolvePartial,
	}
}

// JSON returns an independent copy of the source schema.
func (schema *ContentSchema) JSON() json.RawMessage {
	return bytes.Clone(schema.outputSchema)
}

// resolveContentSchema validates references and compiles one immutable schema
// form for repeated document validation.
func resolveContentSchema(schemaJSON json.RawMessage) (*jsonschema.Resolved, error) {
	var schema jsonschema.Schema

	err := json.Unmarshal(schemaJSON, &schema)
	if err != nil {
		return nil, fmt.Errorf("decode JSON Schema: %w", err)
	}

	// Unmarshal only decodes keywords. Resolve validates the schema, follows
	// references, and prepares the immutable validation state.
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return nil, fmt.Errorf("resolve JSON Schema: %w", err)
	}

	return resolved, nil
}

// buildPartialContentSchema derives an independent schema that retains value
// constraints while making content-presence constraints optional.
func buildPartialContentSchema(schemaJSON json.RawMessage) (json.RawMessage, error) {
	var schema any

	err := json.Unmarshal(schemaJSON, &schema)
	if err != nil {
		return nil, fmt.Errorf("decode partial JSON Schema: %w", err)
	}

	err = makeSchemaPropertiesOptional(schema)
	if err != nil {
		return nil, fmt.Errorf("derive partial JSON Schema: %w", err)
	}

	partialSchema, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("encode partial JSON Schema: %w", err)
	}

	return partialSchema, nil
}

// makeSchemaPropertiesOptional relaxes presence keywords throughout a schema.
// It first preserves oneOf value alternatives whose exclusivity would
// otherwise be invalidated when required fields are removed.
func makeSchemaPropertiesOptional(value any) error {
	resolver, err := newPartialSchemaReferenceResolver(value)
	if err != nil {
		return err
	}

	for _, location := range resolver.partializableSchemas {
		err = relaxPartialOneOf(resolver, location)
		if err != nil {
			return err
		}
	}

	return WalkJSONSchema(value, false, func(value map[string]any) error {
		delete(value, "required")
		delete(value, "dependentRequired")
		delete(value, "minProperties")

		if dependencies, ok := value["dependencies"].(map[string]any); ok {
			for key, dependency := range dependencies {
				if _, requiredProperties := dependency.([]any); requiredProperties {
					delete(dependencies, key)
				}
			}

			if len(dependencies) == 0 {
				delete(value, "dependencies")
			}
		}

		return nil
	})
}

// relaxPartialOneOf changes a presence-discriminated oneOf into anyOf before
// partialization. Scalar and otherwise value-discriminated oneOf constraints
// remain exclusive.
func relaxPartialOneOf(
	resolver partialSchemaReferenceResolver,
	location partialSchemaLocation,
) error {
	schema, objectSchema := location.value.(map[string]any)
	if !objectSchema {
		return nil
	}

	oneOf, hasOneOf := schema["oneOf"].([]any)
	if !hasOneOf {
		return nil
	}

	hasPresenceConstraints, err := partialSchemaHasPresenceConstraints(resolver, location, len(oneOf))
	if err != nil {
		return fmt.Errorf("inspect oneOf presence constraints: %w", err)
	}

	if !hasPresenceConstraints {
		return nil
	}

	delete(schema, "oneOf")

	existingAnyOf, hasAnyOf := schema[jsonSchemaKeywordAnyOf]
	if !hasAnyOf {
		schema[jsonSchemaKeywordAnyOf] = oneOf

		return nil
	}

	delete(schema, jsonSchemaKeywordAnyOf)

	constraints := []any{
		map[string]any{jsonSchemaKeywordAnyOf: existingAnyOf},
		map[string]any{jsonSchemaKeywordAnyOf: oneOf},
	}

	allOf, hasAllOf := schema["allOf"].([]any)
	if hasAllOf {
		schema["allOf"] = append(allOf, constraints...)

		return nil
	}

	schema["allOf"] = constraints

	return nil
}

// partialSchemaHasPresenceConstraints reports whether any oneOf branch depends
// on required properties, including constraints reached through references.
func partialSchemaHasPresenceConstraints(
	resolver partialSchemaReferenceResolver,
	location partialSchemaLocation,
	branchCount int,
) (bool, error) {
	visitedReferences := make(map[string]struct{})

	for index := range branchCount {
		branchPointer := appendJSONSchemaPointer(location.pointer, "oneOf", strconv.Itoa(index))

		branch, exists := resolver.locations[branchPointer]
		if !exists {
			return false, fmt.Errorf(
				"%w: oneOf branch %q is not a schema",
				errJSONSchemaReferenceInvalid,
				branchPointer,
			)
		}

		hasPresenceConstraints, err := partialSchemaLocationHasPresenceConstraints(
			resolver,
			branch,
			visitedReferences,
		)
		if err != nil {
			return false, err
		}

		if hasPresenceConstraints {
			return true, nil
		}
	}

	return false, nil
}

// partialSchemaLocationHasPresenceConstraints walks one schema location and
// its local references without treating predicate-only schemas as content
// presence requirements.
func partialSchemaLocationHasPresenceConstraints(
	resolver partialSchemaReferenceResolver,
	location partialSchemaLocation,
	visitedReferences map[string]struct{},
) (bool, error) {
	schema, objectSchema := location.value.(map[string]any)
	if !objectSchema {
		return false, nil
	}

	if resolver.draft7 {
		if reference, hasReference := schema["$ref"].(string); hasReference && reference != "" {
			return partialSchemaReferenceHasPresenceConstraints(
				resolver,
				location,
				"$ref",
				reference,
				visitedReferences,
			)
		}
	}

	if partialSchemaObjectHasPresenceConstraints(schema) {
		return true, nil
	}

	for _, keyword := range jsonSchemaReferenceKeywords {
		reference, hasReference := schema[keyword].(string)
		if !hasReference || reference == "" {
			continue
		}

		hasPresenceConstraints, err := partialSchemaReferenceHasPresenceConstraints(
			resolver,
			location,
			keyword,
			reference,
			visitedReferences,
		)
		if err != nil {
			return false, err
		}

		if hasPresenceConstraints {
			return true, nil
		}
	}

	hasPresenceConstraints := false

	err := forEachPartialSchemaChildAt(
		schema,
		location.pointer,
		false,
		false,
		func(_ any, childPointer string, _ bool) error {
			if hasPresenceConstraints {
				return nil
			}

			child, exists := resolver.locations[childPointer]
			if !exists {
				return fmt.Errorf(
					"%w: child %q is not a schema",
					errJSONSchemaReferenceInvalid,
					childPointer,
				)
			}

			var childErr error

			hasPresenceConstraints, childErr = partialSchemaLocationHasPresenceConstraints(
				resolver,
				child,
				visitedReferences,
			)

			return childErr
		},
	)

	return hasPresenceConstraints, err
}

// partialSchemaReferenceHasPresenceConstraints resolves a local target once,
// preventing cycles from recursing indefinitely.
func partialSchemaReferenceHasPresenceConstraints(
	resolver partialSchemaReferenceResolver,
	source partialSchemaLocation,
	keyword string,
	reference string,
	visitedReferences map[string]struct{},
) (bool, error) {
	target, err := resolver.resolve(source, reference)
	if err != nil {
		return false, fmt.Errorf("resolve %s: %w", keyword, err)
	}

	if _, visited := visitedReferences[target.pointer]; visited {
		return false, nil
	}

	visitedReferences[target.pointer] = struct{}{}

	return partialSchemaLocationHasPresenceConstraints(resolver, target, visitedReferences)
}

// partialSchemaObjectHasPresenceConstraints recognizes the draft-specific
// keywords whose only purpose is to require object members.
func partialSchemaObjectHasPresenceConstraints(schema map[string]any) bool {
	if required, ok := schema["required"].([]any); ok && len(required) != 0 {
		return true
	}

	if minProperties, ok := schema["minProperties"].(float64); ok && minProperties > 0 {
		return true
	}

	if dependentRequired, ok := schema["dependentRequired"].(map[string]any); ok {
		for _, dependency := range dependentRequired {
			if properties, propertiesOK := dependency.([]any); propertiesOK && len(properties) != 0 {
				return true
			}
		}
	}

	if dependencies, ok := schema["dependencies"].(map[string]any); ok {
		for _, dependency := range dependencies {
			if properties, propertiesOK := dependency.([]any); propertiesOK && len(properties) != 0 {
				return true
			}
		}
	}

	return false
}

// ValidateComplete validates a complete content object and returns its decoded tree.
func (schema *ContentSchema) ValidateComplete(value json.RawMessage) (map[string]any, error) {
	resolved, err := schema.resolveComplete()
	if err != nil {
		return nil, fmt.Errorf("%w: prepare complete form: %w", ErrContentSchemaInvalid, err)
	}

	return schema.validate(value, resolved)
}

// ValidatePartial validates a content object after making presence constraints optional.
func (schema *ContentSchema) ValidatePartial(value json.RawMessage) (map[string]any, error) {
	resolved, err := schema.resolvePartial()
	if err != nil {
		return nil, fmt.Errorf("%w: prepare partial form: %w", ErrContentSchemaInvalid, err)
	}

	return schema.validate(value, resolved)
}

// validate decodes the bounded object before applying a previously resolved
// schema, keeping detailed rejected values out of returned errors.
func (schema *ContentSchema) validate(
	value json.RawMessage,
	resolved *jsonschema.Resolved,
) (map[string]any, error) {
	instance, err := decodeContentDocument(value, schema.maxBytes)
	if err != nil {
		return nil, err
	}

	err = resolved.Validate(instance)
	if err != nil {
		// jsonschema-go includes rejected values in errors; expose an opaque error.
		return nil, errContentDocumentSchemaInvalid
	}

	return instance, nil
}

// decodeContentDocument enforces byte and UTF-8 bounds and accepts exactly one
// JSON object as content.
func decodeContentDocument(value json.RawMessage, maxBytes int) (map[string]any, error) {
	if maxBytes > 0 && len(value) > maxBytes {
		return nil, fmt.Errorf(
			"%w: contains %d bytes, limit is %d",
			errContentDocumentTooLarge,
			len(value),
			maxBytes,
		)
	}

	if !utf8.Valid(value) {
		return nil, errContentDocumentInvalid
	}

	value = bytes.TrimSpace(value)
	if len(value) == 0 {
		return nil, errContentDocumentEmpty
	}

	var instance map[string]any

	err := json.Unmarshal(value, &instance)
	if err != nil {
		return nil, errContentDocumentInvalid
	}

	if instance == nil {
		return nil, errContentDocumentNotObject
	}

	return instance, nil
}
