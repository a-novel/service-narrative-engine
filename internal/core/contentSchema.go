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
	errContentDocumentEmpty     = errors.New("JSON document is empty")
	errContentDocumentNotObject = errors.New("JSON value must be an object")
	errContentDocumentTooLarge  = errors.New("JSON document exceeds the size limit")

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

	makeSchemaPropertiesOptional(schema)

	partialSchema, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("encode partial JSON Schema: %w", err)
	}

	return partialSchema, nil
}

func makeSchemaPropertiesOptional(value any) {
	switch value := value.(type) {
	case map[string]any:
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

		for _, child := range value {
			makeSchemaPropertiesOptional(child)
		}
	case []any:
		for _, child := range value {
			makeSchemaPropertiesOptional(child)
		}
	}
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
		return fmt.Errorf("validate JSON value: %w", err)
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
		return nil, fmt.Errorf("decode JSON value: %w", err)
	}

	if instance == nil {
		return nil, errContentDocumentNotObject
	}

	return instance, nil
}
