package lib

import (
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/samber/lo"
)

// supportedOpenAITypes lists the JSON Schema types supported by OpenAI structured outputs.
// Note: "enum" and "anyOf" are keywords, not types.
// See: https://platform.openai.com/docs/guides/structured-outputs
var supportedOpenAITypes = []string{
	"string",
	"number",
	"integer",
	"boolean",
	"array",
	"object",
	"null",
}

var supportedOpenAIFormats = []string{
	"date-time",
	"time",
	"date",
	"duration",
	"email",
	"hostname",
	"ipv4",
	"ipv6",
	"uuid",
}

// JSONSchemaLLM strips a JSON Schema definition to fit OpenAI supported subset:
// https://platform.openai.com/docs/guides/structured-outputs
func JSONSchemaLLM(src *jsonschema.Schema) bool {
	if src == nil {
		return false
	}

	// AnyOf is a special case.
	if src.AnyOf != nil {
		*src = jsonschema.Schema{AnyOf: lo.Filter(src.AnyOf, func(item *jsonschema.Schema, _ int) bool {
			return JSONSchemaLLM(item)
		})}

		return len(src.AnyOf) > 0
	}

	// Only keep types supported by OpenAI.
	types := append([]string{}, src.Types...)
	types = append(types, src.Type)
	types = lo.Filter(types, func(item string, _ int) bool {
		return lo.Contains(supportedOpenAITypes, item)
	})

	switch len(types) {
	case 0:
		// Ignore values whose type is not supported by OpenAI.
		return false
	case 1:
		src.Type = types[0]
		src.Types = nil
	default:
		src.Type = ""
		src.Types = types
	}

	// Check if the current schema contains unsupported properties that cannot be ignored.
	isSupported := (src.Format == "" || lo.Contains(supportedOpenAIFormats, src.Format)) &&
		src.Const == nil &&
		src.Contains == nil &&
		!src.UniqueItems &&
		!src.Deprecated &&
		!src.ReadOnly &&
		src.PatternProperties == nil &&
		src.AdditionalItems == nil &&
		src.AllOf == nil &&
		src.OneOf == nil &&
		src.PrefixItems == nil
	if !isSupported {
		return false
	}

	isObject := lo.Contains(types, "object")
	isArray := lo.Contains(types, "array")

	// Filter supported object properties.
	if isObject {
		src.Properties = lo.MapValues(src.Properties, func(value *jsonschema.Schema, key string) *jsonschema.Schema {
			if !JSONSchemaLLM(value) {
				return nil
			}

			return value
		})

		// Map all properties and make them required. Properties that were not initially required should be converted
		// to have a union type with null.
		for key, prop := range src.Properties {
			if prop == nil {
				continue
			}

			if !lo.Contains(src.Required, key) {
				if prop.Type != "" {
					prop.Types = []string{prop.Type, "null"}
					prop.Type = ""

					continue
				}

				if !lo.Contains(prop.Types, "null") {
					prop.Types = append(prop.Types, "null")
				}
			}
		}

		src.Required = lo.Keys(src.Properties)
	}

	// Filter supported array items.
	if isArray {
		// Only one of Items or ItemsArray should be set.
		if src.Items != nil {
			if !JSONSchemaLLM(src.Items) {
				return false
			}
		} else if len(src.ItemsArray) > 0 {
			src.ItemsArray = lo.Filter(src.ItemsArray, func(item *jsonschema.Schema, _ int) bool {
				return JSONSchemaLLM(item)
			})
		}
	}

	// Validate that complex types have required content:
	// - Objects must have at least one property
	// - Arrays must have items defined
	// Simple types (string, number, integer, boolean, null) are valid without additional content.
	if isObject && len(src.Properties) == 0 {
		return false
	}

	if isArray && len(src.ItemsArray) == 0 && src.Items == nil {
		return false
	}

	*src = jsonschema.Schema{
		Type:  src.Type,
		Types: src.Types,

		MinLength: src.MinLength,
		MaxLength: src.MaxLength,
		Pattern:   src.Pattern,
		Format:    src.Format,

		MultipleOf:       src.MultipleOf,
		Maximum:          src.Maximum,
		ExclusiveMaximum: src.ExclusiveMaximum,
		Minimum:          src.Minimum,
		ExclusiveMinimum: src.ExclusiveMinimum,

		MinItems: src.MinItems,
		MaxItems: src.MaxItems,
		// Only one of Items or ItemsArray should be set.
		Items:      lo.Ternary(src.Items != nil, src.Items, nil),
		ItemsArray: lo.Ternary(src.Items == nil && len(src.ItemsArray) > 0, src.ItemsArray, nil),

		Properties: src.Properties,
		Required:   src.Required,

		// OpenAI requires all objects to have no additional properties.
		AdditionalProperties: lo.Ternary(isObject, &jsonschema.Schema{Not: &jsonschema.Schema{}}, nil),

		Examples: src.Examples,
	}

	return true
}
