package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
)

const (
	responsesSchemaAdditionalProperties = "additionalProperties"
	responsesSchemaAnyOf                = "anyOf"
	responsesSchemaDefs                 = "$defs"
	responsesSchemaEnum                 = "enum"
	responsesSchemaProperties           = "properties"
	responsesSchemaRequired             = "required"
	responsesSchemaType                 = "type"
	responsesSchemaTypeObject           = "object"
)

var (
	errResponsesSchemaNotObject = errors.New("responses schema must be an object")

	// ErrResponsesSchemaConflict reports a const value excluded by an existing enum.
	ErrResponsesSchemaConflict = errors.New("schema const is excluded by enum")
	// ErrResponsesSchemaUnsupported reports a keyword rejected by strict Responses output.
	ErrResponsesSchemaUnsupported = errors.New("responses schema is unsupported")
)

// ProjectResponsesJSONSchema returns an independent strict Responses API
// projection while leaving the source schema untouched for local validation.
func ProjectResponsesJSONSchema(source json.RawMessage) (json.RawMessage, error) {
	var schema map[string]any

	err := json.Unmarshal(source, &schema)
	if err != nil {
		return nil, fmt.Errorf("decode Responses schema: %w", err)
	}

	if schema == nil {
		return nil, errResponsesSchemaNotObject
	}

	err = WalkJSONSchema(schema, true, projectResponsesSchemaObject)
	if err != nil {
		return nil, err
	}

	projected, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("encode Responses schema: %w", err)
	}

	return projected, nil
}

// projectResponsesSchemaObject removes unsupported annotations and applies the
// strict-object and const rules to one schema node.
func projectResponsesSchemaObject(object map[string]any) error {
	for keyword := range object {
		switch keyword {
		case "$comment",
			"$schema",
			"default",
			"deprecated",
			"examples",
			"maxLength",
			"minLength",
			"readOnly",
			"title",
			"writeOnly":
			delete(object, keyword)
		}
	}

	err := projectResponsesSchemaConst(object)
	if err != nil {
		return err
	}

	err = validateResponsesSchemaKeywords(object)
	if err != nil {
		return err
	}

	enforceResponsesStrictObject(object)

	return nil
}

// validateResponsesSchemaKeywords returns the first unsupported keyword in
// lexical order so definition failures stay deterministic.
func validateResponsesSchemaKeywords(object map[string]any) error {
	unsupported := make([]string, 0)

	for keyword := range object {
		if !responsesSchemaKeywordSupported(keyword) {
			unsupported = append(unsupported, keyword)
		}
	}

	if len(unsupported) == 0 {
		return nil
	}

	slices.Sort(unsupported)

	return fmt.Errorf(
		"%w: JSON Schema keyword %q",
		ErrResponsesSchemaUnsupported,
		unsupported[0],
	)
}

// responsesSchemaKeywordSupported reports whether strict Responses output
// accepts a schema keyword retained by the projection.
func responsesSchemaKeywordSupported(keyword string) bool {
	switch keyword {
	case responsesSchemaDefs,
		"$ref",
		responsesSchemaAdditionalProperties,
		responsesSchemaAnyOf,
		"description",
		responsesSchemaEnum,
		"exclusiveMaximum",
		"exclusiveMinimum",
		"format",
		"items",
		"maxItems",
		"maximum",
		"minItems",
		"minimum",
		"multipleOf",
		"pattern",
		responsesSchemaProperties,
		responsesSchemaRequired,
		responsesSchemaType:
		return true
	default:
		return false
	}
}

// enforceResponsesStrictObject closes an object schema and requires every
// declared property, including objects reached through unions and references.
func enforceResponsesStrictObject(object map[string]any) {
	if !schemaTypeIncludesObject(object[responsesSchemaType]) {
		return
	}

	properties, _ := object[responsesSchemaProperties].(map[string]any)
	if properties == nil {
		properties = make(map[string]any)
	}

	required := make([]string, 0, len(properties))
	for property := range properties {
		required = append(required, property)
	}

	slices.Sort(required)

	object[responsesSchemaAdditionalProperties] = false
	object[responsesSchemaProperties] = properties
	object[responsesSchemaRequired] = required
}

// schemaTypeIncludesObject recognizes scalar and union encodings of the object type.
func schemaTypeIncludesObject(value any) bool {
	if value == responsesSchemaTypeObject {
		return true
	}

	types, typesOK := value.([]any)
	if !typesOK {
		return false
	}

	for _, schemaType := range types {
		if schemaType == responsesSchemaTypeObject {
			return true
		}
	}

	return false
}

// projectResponsesSchemaConst rewrites const as the one-value enum accepted by
// strict Responses output.
func projectResponsesSchemaConst(object map[string]any) error {
	constant, hasConstant := object["const"]
	if !hasConstant {
		return nil
	}

	enum, hasEnum := object[responsesSchemaEnum].([]any)
	if hasEnum && !schemaEnumContains(enum, constant) {
		return ErrResponsesSchemaConflict
	}

	object[responsesSchemaEnum] = []any{constant}
	delete(object, "const")

	return nil
}

// schemaEnumContains compares arbitrary decoded JSON values without assuming a scalar type.
func schemaEnumContains(enum []any, expected any) bool {
	for _, candidate := range enum {
		if reflect.DeepEqual(candidate, expected) {
			return true
		}
	}

	return false
}
