package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

const (
	contentShortMaxCharacters   = 128
	contentRegularMaxCharacters = 32 * 1024
	contentDocumentMaxBytes     = 5 * 1024 * 1024
)

var (
	errContentDocumentEmpty     = errors.New("JSON document is empty")
	errContentDocumentNotObject = errors.New("JSON value must be an object")
	errContentDocumentTooLarge  = errors.New("JSON document exceeds the size limit")
)

var ideaOutputSchema = json.RawMessage(fmt.Sprintf(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["title", "genre", "seed"],
  "properties": {
    "title": {
      "type": "string",
      "minLength": 1,
      "maxLength": %[1]d,
      "pattern": "\\S"
    },
    "genre": {
      "type": "string",
      "minLength": 1,
      "maxLength": %[1]d,
      "pattern": "\\S"
    },
    "seed": {
      "type": "string",
      "minLength": 1,
      "maxLength": %[2]d,
      "pattern": "\\S"
    }
  }
}`, contentShortMaxCharacters, contentRegularMaxCharacters))

var manuscriptOutputSchema = json.RawMessage(fmt.Sprintf(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["blocks"],
  "properties": {
    "blocks": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["type", "text", "marks"],
        "properties": {
          "type": {"const": "text"},
          "text": {
            "type": "string",
            "minLength": 1,
            "maxLength": %[1]d,
            "pattern": "\\S"
          },
          "marks": {
            "type": "array",
            "items": {
              "type": "object",
              "additionalProperties": false,
              "required": ["type", "start", "end"],
              "properties": {
                "type": {
                  "type": "string",
                  "enum": ["bold", "italic", "underline", "strikethrough"]
                },
                "start": {
                  "type": "integer",
                  "minimum": 0,
                  "maximum": %[1]d
                },
                "end": {
                  "type": "integer",
                  "minimum": 1,
                  "maximum": %[1]d
                }
              }
            }
          }
        }
      }
    }
  }
}`, contentRegularMaxCharacters))

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
