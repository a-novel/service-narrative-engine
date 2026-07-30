package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

const contentDocumentMaxBytes = 5 * 1024 * 1024

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
	return walkJSONSchema(value, false, func(value map[string]any) error {
		relaxPartialOneOf(value)

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

func relaxPartialOneOf(schema map[string]any) {
	oneOf, hasOneOf := schema["oneOf"].([]any)
	if !hasOneOf || !partialSchemaHasPresenceConstraints(oneOf) {
		return
	}

	delete(schema, "oneOf")

	existingAnyOf, hasAnyOf := schema[jsonSchemaKeywordAnyOf]
	if !hasAnyOf {
		schema[jsonSchemaKeywordAnyOf] = oneOf

		return
	}

	delete(schema, jsonSchemaKeywordAnyOf)

	constraints := []any{
		map[string]any{jsonSchemaKeywordAnyOf: existingAnyOf},
		map[string]any{jsonSchemaKeywordAnyOf: oneOf},
	}

	allOf, hasAllOf := schema["allOf"].([]any)
	if hasAllOf {
		schema["allOf"] = append(allOf, constraints...)

		return
	}

	schema["allOf"] = constraints
}

func partialSchemaHasPresenceConstraints(value any) bool {
	found := false

	_ = walkJSONSchemaValue(value, false, func(schema map[string]any) error {
		if partialSchemaObjectHasPresenceConstraints(schema) {
			found = true
		}

		return nil
	})

	return found
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

	return nil
}

func decodeContentDocument(value json.RawMessage) (map[string]any, error) {
	if len(value) > contentDocumentMaxBytes {
		return nil, fmt.Errorf(
			"%w: contains %d bytes, limit is %d",
			errContentDocumentTooLarge,
			len(value),
			contentDocumentMaxBytes,
		)
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
