package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

var (
	errContentDocumentEmpty         = errors.New("json document is empty")
	errContentDocumentInvalid       = errors.New("json document is invalid")
	errContentDocumentNotObject     = errors.New("json value must be an object")
	errContentDocumentSchemaInvalid = errors.New("json document does not match its schema")
	errContentDocumentTooLarge      = errors.New("json document exceeds the size limit")

	ideaOutputSchema       = schemas.Idea
	manuscriptOutputSchema = schemas.Manuscript
)

type contentSchemaDefinition struct {
	OutputSchema          json.RawMessage
	resolvedSchema        *jsonschema.Resolved
	resolvedPartialSchema *jsonschema.Resolved
	validateSemantics     func(map[string]any) error
}

func loadContentSchema(outputSchema json.RawMessage) (*contentSchemaDefinition, error) {
	resolvedSchema, err := resolveContentSchema(outputSchema)
	if err != nil {
		return nil, err
	}

	partialSchema, err := buildPartialContentSchema(outputSchema)
	if err != nil {
		return nil, err
	}

	resolvedPartialSchema, err := resolveContentSchema(partialSchema)
	if err != nil {
		return nil, err
	}

	return &contentSchemaDefinition{
		OutputSchema:          outputSchema,
		resolvedSchema:        resolvedSchema,
		resolvedPartialSchema: resolvedPartialSchema,
	}, nil
}

func resolveContentSchema(schemaJSON json.RawMessage) (*jsonschema.Resolved, error) {
	var schema jsonschema.Schema

	err := json.Unmarshal(schemaJSON, &schema)
	if err != nil {
		return nil, fmt.Errorf("decode JSON Schema: %w", err)
	}

	resolved, err := schema.Resolve(nil)
	if err != nil {
		return nil, fmt.Errorf("resolve JSON Schema: %w", err)
	}

	return resolved, nil
}

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

	return walkJSONSchema(value, false, func(value map[string]any) error {
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

func (definition *contentSchemaDefinition) validateComplete(value json.RawMessage) error {
	return definition.validate(value, definition.resolvedSchema)
}

func (definition *contentSchemaDefinition) validatePartial(value json.RawMessage) error {
	return definition.validate(value, definition.resolvedPartialSchema)
}

func (definition *contentSchemaDefinition) validate(
	value json.RawMessage,
	schema *jsonschema.Resolved,
) error {
	instance, err := decodeContentDocument(value)
	if err != nil {
		return err
	}

	err = schema.Validate(instance)
	if err != nil {
		// jsonschema-go includes rejected instance values in its errors. Keep
		// project content out of returned errors and trace exception messages.
		return errContentDocumentSchemaInvalid
	}

	if definition.validateSemantics == nil {
		return nil
	}

	return definition.validateSemantics(instance)
}

func decodeContentDocument(value json.RawMessage) (map[string]any, error) {
	if len(value) > schemas.ContentDocumentMaxBytes {
		return nil, fmt.Errorf(
			"%w: contains %d bytes, limit is %d",
			errContentDocumentTooLarge,
			len(value),
			schemas.ContentDocumentMaxBytes,
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
