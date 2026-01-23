package lib_test

import (
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestJSONSchemaLLM(t *testing.T) {
	t.Parallel()

	t.Run("NilInput", func(t *testing.T) {
		t.Parallel()

		result := lib.JSONSchemaLLM(nil)
		require.False(t, result)
	})

	t.Run("AnyOf", func(t *testing.T) {
		t.Parallel()

		t.Run("ValidItems", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				AnyOf: []*jsonschema.Schema{
					{
						Type:       "object",
						Properties: map[string]*jsonschema.Schema{"foo": {Type: "string"}},
					},
					{
						Type:       "object",
						Properties: map[string]*jsonschema.Schema{"bar": {Type: "integer"}},
					},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Len(t, schema.AnyOf, 2)
		})

		t.Run("FilterInvalidItems", func(t *testing.T) {
			t.Parallel()

			constVal := any("invalid")
			schema := &jsonschema.Schema{
				AnyOf: []*jsonschema.Schema{
					{
						Type:       "object",
						Properties: map[string]*jsonschema.Schema{"foo": {Type: "string"}},
					},
					{
						Type:       "object",
						Const:      &constVal, // Unsupported property
						Properties: map[string]*jsonschema.Schema{"bar": {Type: "integer"}},
					},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Len(t, schema.AnyOf, 1)
		})

		t.Run("AllItemsInvalid", func(t *testing.T) {
			t.Parallel()

			constVal := any("invalid")
			schema := &jsonschema.Schema{
				AnyOf: []*jsonschema.Schema{
					{
						Type:       "object",
						Const:      &constVal,
						Properties: map[string]*jsonschema.Schema{"foo": {Type: "string"}},
					},
					{
						Type:       "object",
						Deprecated: true,
						Properties: map[string]*jsonschema.Schema{"bar": {Type: "integer"}},
					},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})
	})

	t.Run("Types", func(t *testing.T) {
		t.Parallel()

		t.Run("SingleSupportedType", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:       "object",
				Properties: map[string]*jsonschema.Schema{"foo": {Type: "string"}},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Equal(t, "object", schema.Type)
			require.Nil(t, schema.Types)
		})

		t.Run("MultipleSupportedTypes", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Types: []string{"object", "null"},
				Properties: map[string]*jsonschema.Schema{
					"foo": {Type: "string"},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Empty(t, schema.Type)
			require.ElementsMatch(t, []string{"object", "null"}, schema.Types)
		})

		t.Run("UnsupportedType", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "unsupported",
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})

		t.Run("MixedTypes", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Types: []string{"object", "unsupported", "null"},
				Properties: map[string]*jsonschema.Schema{
					"foo": {Type: "string"},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Empty(t, schema.Type)
			require.ElementsMatch(t, []string{"object", "null"}, schema.Types)
		})

		t.Run("TypeAndTypesField", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:  "object",
				Types: []string{"null"},
				Properties: map[string]*jsonschema.Schema{
					"foo": {Type: "string"},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Empty(t, schema.Type)
			require.ElementsMatch(t, []string{"object", "null"}, schema.Types)
		})
	})

	t.Run("UnsupportedProperties", func(t *testing.T) {
		t.Parallel()

		t.Run("Const", func(t *testing.T) {
			t.Parallel()

			constVal := any("value")
			schema := &jsonschema.Schema{
				Type:       "object",
				Const:      &constVal,
				Properties: map[string]*jsonschema.Schema{"foo": {Type: "string"}},
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})

		t.Run("Contains", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:       "object",
				Contains:   &jsonschema.Schema{Type: "string"},
				Properties: map[string]*jsonschema.Schema{"foo": {Type: "string"}},
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})

		t.Run("UniqueItems", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:        "object",
				UniqueItems: true,
				Properties:  map[string]*jsonschema.Schema{"foo": {Type: "string"}},
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})

		t.Run("Deprecated", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:       "object",
				Deprecated: true,
				Properties: map[string]*jsonschema.Schema{"foo": {Type: "string"}},
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})

		t.Run("ReadOnly", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:       "object",
				ReadOnly:   true,
				Properties: map[string]*jsonschema.Schema{"foo": {Type: "string"}},
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})

		t.Run("PatternProperties", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:              "object",
				PatternProperties: map[string]*jsonschema.Schema{"^foo": {Type: "string"}},
				Properties:        map[string]*jsonschema.Schema{"bar": {Type: "string"}},
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})

		t.Run("AdditionalItems", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:            "object",
				AdditionalItems: &jsonschema.Schema{Type: "string"},
				Properties:      map[string]*jsonschema.Schema{"foo": {Type: "string"}},
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})

		t.Run("AllOf", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:       "object",
				AllOf:      []*jsonschema.Schema{{Type: "string"}},
				Properties: map[string]*jsonschema.Schema{"foo": {Type: "string"}},
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})

		t.Run("OneOf", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:       "object",
				OneOf:      []*jsonschema.Schema{{Type: "string"}},
				Properties: map[string]*jsonschema.Schema{"foo": {Type: "string"}},
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})

		t.Run("PrefixItems", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:        "object",
				PrefixItems: []*jsonschema.Schema{{Type: "string"}},
				Properties:  map[string]*jsonschema.Schema{"foo": {Type: "string"}},
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})
	})

	t.Run("Format", func(t *testing.T) {
		t.Parallel()

		supportedFormats := []string{
			"date-time", "time", "date", "duration",
			"email", "hostname", "ipv4", "ipv6", "uuid",
		}

		for _, format := range supportedFormats {
			t.Run("Supported/"+format, func(t *testing.T) {
				t.Parallel()

				schema := &jsonschema.Schema{
					Type:       "object",
					Format:     format,
					Properties: map[string]*jsonschema.Schema{"foo": {Type: "string"}},
				}

				result := lib.JSONSchemaLLM(schema)
				require.True(t, result)
			})
		}

		t.Run("UnsupportedFormat", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:       "object",
				Format:     "uri",
				Properties: map[string]*jsonschema.Schema{"foo": {Type: "string"}},
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})

		t.Run("EmptyFormat", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:       "object",
				Format:     "",
				Properties: map[string]*jsonschema.Schema{"foo": {Type: "string"}},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
		})
	})

	t.Run("Object", func(t *testing.T) {
		t.Parallel()

		t.Run("NoProperties", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "object",
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})

		t.Run("FilterInvalidProperties", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"valid":   {Type: "string"},
					"invalid": {Type: "unsupported"},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Contains(t, schema.Properties, "valid")
			require.Nil(t, schema.Properties["invalid"])
		})

		t.Run("RequiredPropertiesStayUnchanged", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"foo": {Type: "string"},
				},
				Required: []string{"foo"},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Equal(t, "string", schema.Properties["foo"].Type)
			require.Nil(t, schema.Properties["foo"].Types)
		})

		t.Run("NonRequiredPropertiesWithTypeBecomesNullable", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"optional": {Type: "string"},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Empty(t, schema.Properties["optional"].Type)
			require.ElementsMatch(t, []string{"string", "null"}, schema.Properties["optional"].Types)
			require.Contains(t, schema.Required, "optional")
		})

		t.Run("NonRequiredPropertiesWithTypesBecomesNullable", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"optional": {Types: []string{"string", "integer"}},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.ElementsMatch(t, []string{"string", "integer", "null"}, schema.Properties["optional"].Types)
		})

		t.Run("NonRequiredPropertiesAlreadyNullable", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"optional": {Types: []string{"string", "null"}},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.ElementsMatch(t, []string{"string", "null"}, schema.Properties["optional"].Types)
		})

		t.Run("AllPropertiesBecomesRequired", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"foo": {Type: "string"},
					"bar": {Type: "integer"},
				},
				Required: []string{"foo"},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.ElementsMatch(t, []string{"foo", "bar"}, schema.Required)
		})

		t.Run("AdditionalPropertiesFalse", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"foo": {Type: "string"},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.NotNil(t, schema.AdditionalProperties)
			require.NotNil(t, schema.AdditionalProperties.Not)
		})

		t.Run("NestedObject", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"nested": {
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"inner": {Type: "string"},
						},
					},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.NotNil(t, schema.Properties["nested"])
			require.Contains(t, schema.Properties["nested"].Properties, "inner")
		})
	})

	t.Run("Array", func(t *testing.T) {
		t.Parallel()

		t.Run("NoItems", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "array",
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})

		t.Run("WithItems", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:  "array",
				Items: &jsonschema.Schema{Type: "string"},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.NotNil(t, schema.Items)
		})

		t.Run("WithInvalidItems", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:  "array",
				Items: &jsonschema.Schema{Type: "unsupported"},
			}

			result := lib.JSONSchemaLLM(schema)
			require.False(t, result)
		})

		t.Run("WithItemsArray", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "array",
				ItemsArray: []*jsonschema.Schema{
					{Type: "string"},
					{Type: "integer"},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Len(t, schema.ItemsArray, 2)
		})

		t.Run("FilterInvalidItemsArray", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "array",
				ItemsArray: []*jsonschema.Schema{
					{Type: "string"},
					{Type: "unsupported"},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Len(t, schema.ItemsArray, 1)
		})

		t.Run("WithObjectItems", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "array",
				Items: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"foo": {Type: "string"},
					},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.NotNil(t, schema.Items)
			require.Contains(t, schema.Items.Properties, "foo")
		})

		t.Run("NoAdditionalPropertiesForNonObject", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:  "array",
				Items: &jsonschema.Schema{Type: "string"},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Nil(t, schema.AdditionalProperties)
		})
	})

	t.Run("PreservedFields", func(t *testing.T) {
		t.Parallel()

		t.Run("StringConstraints", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"foo": {
						Type:      "string",
						MinLength: lo.ToPtr(5),
						MaxLength: lo.ToPtr(10),
						Pattern:   "^[a-z]+$",
					},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Equal(t, lo.ToPtr(5), schema.Properties["foo"].MinLength)
			require.Equal(t, lo.ToPtr(10), schema.Properties["foo"].MaxLength)
			require.Equal(t, "^[a-z]+$", schema.Properties["foo"].Pattern)
		})

		t.Run("NumberConstraints", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"foo": {
						Type:             "number",
						MultipleOf:       lo.ToPtr(2.5),
						Maximum:          lo.ToPtr(100.0),
						ExclusiveMaximum: lo.ToPtr(101.0),
						Minimum:          lo.ToPtr(0.0),
						ExclusiveMinimum: lo.ToPtr(-1.0),
					},
				},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Equal(t, lo.ToPtr(2.5), schema.Properties["foo"].MultipleOf)
			require.Equal(t, lo.ToPtr(100.0), schema.Properties["foo"].Maximum)
			require.Equal(t, lo.ToPtr(101.0), schema.Properties["foo"].ExclusiveMaximum)
			require.Equal(t, lo.ToPtr(0.0), schema.Properties["foo"].Minimum)
			require.Equal(t, lo.ToPtr(-1.0), schema.Properties["foo"].ExclusiveMinimum)
		})

		t.Run("ArrayConstraints", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type:     "array",
				Items:    &jsonschema.Schema{Type: "string"},
				MinItems: lo.ToPtr(1),
				MaxItems: lo.ToPtr(10),
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Equal(t, lo.ToPtr(1), schema.MinItems)
			require.Equal(t, lo.ToPtr(10), schema.MaxItems)
		})

		t.Run("Examples", func(t *testing.T) {
			t.Parallel()

			schema := &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"foo": {Type: "string"},
				},
				Examples: []any{"example1", "example2"},
			}

			result := lib.JSONSchemaLLM(schema)
			require.True(t, result)
			require.Equal(t, []any{"example1", "example2"}, schema.Examples)
		})
	})
}
