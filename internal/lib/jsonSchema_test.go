package lib_test

import (
	"encoding/json"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

// =============================================================================
// SchemaType
// =============================================================================

func TestSchemaTypeMarshalJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		input  lib.SchemaType
		expect string
	}{
		{
			name:   "Empty",
			input:  lib.NewSchemaType(),
			expect: "null",
		},
		{
			name:   "Single",
			input:  lib.NewSchemaType("string"),
			expect: `"string"`,
		},
		{
			name:   "Multiple",
			input:  lib.NewSchemaType("string", "null"),
			expect: `["string","null"]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := json.Marshal(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expect, string(actual))
		})
	}
}

func TestSchemaTypeUnmarshalJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		input     string
		expect    lib.SchemaType
		expectErr error
	}{
		{
			name:   "String",
			input:  `"string"`,
			expect: lib.NewSchemaType("string"),
		},
		{
			name:   "Array",
			input:  `["string","null"]`,
			expect: lib.NewSchemaType("string", "null"),
		},
		{
			name:   "ArraySingleElement",
			input:  `["object"]`,
			expect: lib.NewSchemaType("object"),
		},
		{
			name:      "InvalidNumber",
			input:     `123`,
			expectErr: lib.ErrInvalidSchemaFormat,
		},
		{
			name:      "InvalidBool",
			input:     `true`,
			expectErr: lib.ErrInvalidSchemaFormat,
		},
		{
			name:      "ArrayWithNonString",
			input:     `["string", 123]`,
			expectErr: lib.ErrInvalidSchemaFormat,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var result lib.SchemaType
			err := json.Unmarshal([]byte(tc.input), &result)

			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expect, result)
			}
		})
	}
}

func TestSchemaTypeMarshalYAML(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   lib.SchemaType
		wantRaw any // raw primitive that yaml.Marshal should have been called with
	}{
		{
			name:    "Empty",
			input:   lib.NewSchemaType(),
			wantRaw: nil,
		},
		{
			name:    "Single",
			input:   lib.NewSchemaType("string"),
			wantRaw: "string",
		},
		{
			name:    "Multiple",
			input:   lib.NewSchemaType("string", "number"),
			wantRaw: []string{"string", "number"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expected, err := yaml.Marshal(tc.wantRaw)
			require.NoError(t, err)

			actual, err := tc.input.MarshalYAML()
			require.NoError(t, err)

			require.Equal(t, expected, actual)
		})
	}
}

func TestSchemaTypeUnmarshalYAML(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		input     []byte
		expect    lib.SchemaType
		expectErr error
	}{
		{
			name:   "String",
			input:  []byte("string\n"),
			expect: lib.NewSchemaType("string"),
		},
		{
			name:   "Array",
			input:  []byte("- string\n- number\n"),
			expect: lib.NewSchemaType("string", "number"),
		},
		{
			name:      "InvalidInteger",
			input:     []byte("123\n"),
			expectErr: lib.ErrInvalidSchemaFormat,
		},
		{
			name:      "ArrayWithNonString",
			input:     []byte("- string\n- 123\n"),
			expectErr: lib.ErrInvalidSchemaFormat,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var result lib.SchemaType
			err := result.UnmarshalYAML(tc.input)

			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expect, result)
			}
		})
	}
}

// =============================================================================
// SchemaAdditionalProperties
// =============================================================================

func TestSchemaAdditionalPropertiesMarshalJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		input  *lib.SchemaAdditionalProperties
		expect string
	}{
		{
			name:   "BoolTrue",
			input:  lib.AdditionalPropertiesBool(true),
			expect: "true",
		},
		{
			name:   "BoolFalse",
			input:  lib.AdditionalPropertiesBool(false),
			expect: "false",
		},
		{
			name: "Schema",
			input: lib.AdditionalPropertiesSchema(&lib.RawSchema{
				Type: lib.NewSchemaType("string"),
			}),
			expect: `{"type":"string"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := json.Marshal(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expect, string(actual))
		})
	}
}

func TestSchemaAdditionalPropertiesUnmarshalJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		input        string
		expectReMarshal string // re-marshal the result to verify what was stored
	}{
		{
			name:         "BoolTrue",
			input:        "true",
			expectReMarshal: "true",
		},
		{
			name:         "BoolFalse",
			input:        "false",
			expectReMarshal: "false",
		},
		{
			name:         "Schema",
			input:        `{"type":"string"}`,
			expectReMarshal: `{"type":"string"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var result lib.SchemaAdditionalProperties
			err := json.Unmarshal([]byte(tc.input), &result)
			require.NoError(t, err)

			// Re-marshal to confirm the correct value was stored.
			actual, err := json.Marshal(&result)
			require.NoError(t, err)
			require.Equal(t, tc.expectReMarshal, string(actual))
		})
	}
}

// =============================================================================
// RawSchema.Validate
// =============================================================================

func TestRawSchemaValidate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		schema    lib.RawSchema
		flags     []lib.SchemaValidateFlag
		expectErr error
	}{
		// -------------------------------------------------------------------------
		// Base validation (no flags)
		// -------------------------------------------------------------------------
		{
			name:      "NoType",
			schema:    lib.RawSchema{},
			expectErr: lib.ErrInvalidSchema,
		},
		{
			name: "ValidType",
			schema: lib.RawSchema{
				Type: lib.NewSchemaType("object"),
			},
		},
		{
			name: "ValidTypeNested",
			schema: lib.RawSchema{
				Type: lib.NewSchemaType("object"),
				Properties: map[string]lib.RawSchema{
					"field1": {Type: lib.NewSchemaType("string")},
				},
			},
		},
		{
			name: "NestedMissingType",
			schema: lib.RawSchema{
				Type: lib.NewSchemaType("object"),
				Properties: map[string]lib.RawSchema{
					"bad": {},
				},
			},
			expectErr: lib.ErrInvalidSchema,
		},
		{
			name: "NoGenerateSkipsValidation",
			schema: lib.RawSchema{
				NoGenerate: true,
				Type:       lib.NewSchemaType("custom-type"),
			},
			flags: []lib.SchemaValidateFlag{lib.SchemaValidateFlagGenerate},
		},

		// -------------------------------------------------------------------------
		// SchemaValidateFlagGenerate
		// -------------------------------------------------------------------------
		{
			name: "Generate_ValidType",
			schema: lib.RawSchema{
				Type: lib.NewSchemaType("string"),
			},
			flags: []lib.SchemaValidateFlag{lib.SchemaValidateFlagGenerate},
		},
		{
			name: "Generate_ValidTypeObject",
			schema: lib.RawSchema{
				Type: lib.NewSchemaType("object"),
			},
			flags: []lib.SchemaValidateFlag{lib.SchemaValidateFlagGenerate},
		},
		{
			name: "Generate_ValidUnionWithNull",
			schema: lib.RawSchema{
				Type: lib.NewSchemaType("string", "null"),
			},
			flags: []lib.SchemaValidateFlag{lib.SchemaValidateFlagGenerate},
		},
		{
			name: "Generate_ThreeTypes",
			schema: lib.RawSchema{
				Type: lib.NewSchemaType("string", "number", "null"),
			},
			flags:     []lib.SchemaValidateFlag{lib.SchemaValidateFlagGenerate},
			expectErr: lib.ErrInvalidSchema,
		},
		{
			name: "Generate_TwoTypesWithoutNull",
			schema: lib.RawSchema{
				Type: lib.NewSchemaType("string", "number"),
			},
			flags:     []lib.SchemaValidateFlag{lib.SchemaValidateFlagGenerate},
			expectErr: lib.ErrInvalidSchema,
		},
		{
			name: "Generate_UnsupportedType",
			schema: lib.RawSchema{
				Type: lib.NewSchemaType("unsupported"),
			},
			flags:     []lib.SchemaValidateFlag{lib.SchemaValidateFlagGenerate},
			expectErr: lib.ErrInvalidSchema,
		},
		{
			name: "Generate_ValidFormat",
			schema: lib.RawSchema{
				Type:   lib.NewSchemaType("string"),
				Format: "date-time",
			},
			flags: []lib.SchemaValidateFlag{lib.SchemaValidateFlagGenerate},
		},
		{
			name: "Generate_InvalidFormat",
			schema: lib.RawSchema{
				Type:   lib.NewSchemaType("string"),
				Format: "unsupported-format",
			},
			flags:     []lib.SchemaValidateFlag{lib.SchemaValidateFlagGenerate},
			expectErr: lib.ErrInvalidSchema,
		},
		{
			name: "Generate_AllFormats",
			schema: lib.RawSchema{
				Type:   lib.NewSchemaType("string"),
				Format: "uuid",
			},
			flags: []lib.SchemaValidateFlag{lib.SchemaValidateFlagGenerate},
		},
		{
			name: "Generate_NestedValidSchema",
			schema: lib.RawSchema{
				Type: lib.NewSchemaType("object"),
				Properties: map[string]lib.RawSchema{
					"name": {Type: lib.NewSchemaType("string")},
					"age":  {Type: lib.NewSchemaType("integer")},
				},
			},
			flags: []lib.SchemaValidateFlag{lib.SchemaValidateFlagGenerate},
		},
		{
			name: "Generate_NestedInvalidType",
			schema: lib.RawSchema{
				Type: lib.NewSchemaType("object"),
				Properties: map[string]lib.RawSchema{
					"field": {Type: lib.NewSchemaType("unsupported")},
				},
			},
			flags:     []lib.SchemaValidateFlag{lib.SchemaValidateFlagGenerate},
			expectErr: lib.ErrInvalidSchema,
		},
		{
			name: "Generate_NestedNoGenerateSkipsCheck",
			schema: lib.RawSchema{
				Type: lib.NewSchemaType("object"),
				Properties: map[string]lib.RawSchema{
					"field": {
						NoGenerate: true,
						Type:       lib.NewSchemaType("custom-type"),
					},
				},
			},
			flags: []lib.SchemaValidateFlag{lib.SchemaValidateFlagGenerate},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.schema.Validate(tc.flags...)

			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// BuildSchema
// =============================================================================

func TestBuildSchema(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		schema    *lib.RawSchema
		flags     []lib.SchemaBuildFlag
		expectErr error
	}{
		{
			name: "ValidMinimalSchema",
			schema: &lib.RawSchema{
				Type: lib.NewSchemaType("string"),
			},
		},
		{
			name: "ValidObjectSchema",
			schema: &lib.RawSchema{
				Type: lib.NewSchemaType("object"),
				Properties: map[string]lib.RawSchema{
					"name": {Type: lib.NewSchemaType("string")},
				},
				Required: []string{"name"},
			},
		},
		{
			name: "InvalidNoType",
			schema: &lib.RawSchema{
				Description: "schema with no type",
			},
			expectErr: lib.ErrInvalidSchema,
		},
		{
			name: "InvalidUnsupportedType",
			schema: &lib.RawSchema{
				Type: lib.NewSchemaType("custom-type"),
			},
			expectErr: lib.ErrInvalidSchema,
		},
		{
			name: "NoGenerateSchemaWithPartialFlag",
			schema: &lib.RawSchema{
				NoGenerate: true,
				Type:       lib.NewSchemaType("custom-type"),
			},
			flags: []lib.SchemaBuildFlag{lib.SchemaBuildFlagPartial},
		},
		{
			name: "AIFlag_NoGenerateTopLevel",
			schema: &lib.RawSchema{
				NoGenerate: true,
				Type:       lib.NewSchemaType("object"),
			},
			flags:     []lib.SchemaBuildFlag{lib.SchemaBuildFlagAI},
			expectErr: lib.ErrSchemaNotGeneratable,
		},
		{
			name: "AIFlag_ValidSchema",
			schema: &lib.RawSchema{
				Type: lib.NewSchemaType("object"),
				Properties: map[string]lib.RawSchema{
					"title": {Type: lib.NewSchemaType("string")},
				},
			},
			flags: []lib.SchemaBuildFlag{lib.SchemaBuildFlagAI},
		},
		{
			name: "NestedInvalidType",
			schema: &lib.RawSchema{
				Type: lib.NewSchemaType("object"),
				Properties: map[string]lib.RawSchema{
					"bad": {Type: lib.NewSchemaType("unsupported")},
				},
			},
			expectErr: lib.ErrInvalidSchema,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resolved, err := lib.BuildSchema(tc.schema, tc.flags...)

			if tc.expectErr != nil {
				require.ErrorIs(t, err, tc.expectErr)
				require.Nil(t, resolved)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resolved)
			}
		})
	}
}

// TestBuildSchemaAIFlagPrepares guards against regressions in the AI flag
// preparation logic, specifically:
//
//  1. additionalProperties:false must appear in the output sent to OpenAI
//     (previously the in-place change was lost due to a local pointer
//     reassignment, so the original schema was marshaled unchanged).
//  2. The properties loop must iterate over the *original* properties, not the
//     newly-created (empty) struct (previously all properties were silently
//     dropped).
//  3. Properties marked NoGenerate must be excluded from the AI schema.
func TestBuildSchemaAIFlagPrepares(t *testing.T) {
	t.Parallel()

	t.Run("SetsAdditionalPropertiesFalse", func(t *testing.T) {
		t.Parallel()

		schema := &lib.RawSchema{
			Type: lib.NewSchemaType("object"),
			Properties: map[string]lib.RawSchema{
				"title": {Type: lib.NewSchemaType("string")},
			},
		}

		_, err := lib.BuildSchema(schema, lib.SchemaBuildFlagAI)
		require.NoError(t, err)

		// BuildSchema modifies the schema in place; marshal it to observe the
		// value that was actually sent to the JSON schema resolver.
		data, err := json.Marshal(schema)
		require.NoError(t, err)

		var out map[string]any
		require.NoError(t, json.Unmarshal(data, &out))

		require.Equal(t, false, out["additionalProperties"],
			"additionalProperties must be false for OpenAI structured outputs")
	})

	t.Run("PreservesGeneratableProperties", func(t *testing.T) {
		t.Parallel()

		schema := &lib.RawSchema{
			Type: lib.NewSchemaType("object"),
			Properties: map[string]lib.RawSchema{
				"title":   {Type: lib.NewSchemaType("string")},
				"summary": {Type: lib.NewSchemaType("string")},
			},
		}

		_, err := lib.BuildSchema(schema, lib.SchemaBuildFlagAI)
		require.NoError(t, err)

		require.Contains(t, schema.Properties, "title",
			"generatable property 'title' must be preserved")
		require.Contains(t, schema.Properties, "summary",
			"generatable property 'summary' must be preserved")
	})

	t.Run("FiltersNoGenerateProperties", func(t *testing.T) {
		t.Parallel()

		schema := &lib.RawSchema{
			Type: lib.NewSchemaType("object"),
			Properties: map[string]lib.RawSchema{
				"keep":   {Type: lib.NewSchemaType("string")},
				"hidden": {Type: lib.NewSchemaType("string"), NoGenerate: true},
			},
		}

		_, err := lib.BuildSchema(schema, lib.SchemaBuildFlagAI)
		require.NoError(t, err)

		require.Contains(t, schema.Properties, "keep",
			"generatable property 'keep' must be preserved")
		require.NotContains(t, schema.Properties, "hidden",
			"NoGenerate property 'hidden' must be excluded from AI schema")
	})

	t.Run("NestedObjectsAlsoGetAdditionalPropertiesFalse", func(t *testing.T) {
		t.Parallel()

		nested := &lib.RawSchema{
			Type: lib.NewSchemaType("object"),
			Properties: map[string]lib.RawSchema{
				"value": {Type: lib.NewSchemaType("string")},
			},
		}
		schema := &lib.RawSchema{
			Type: lib.NewSchemaType("object"),
			Properties: map[string]lib.RawSchema{
				"inner": *nested,
			},
		}

		_, err := lib.BuildSchema(schema, lib.SchemaBuildFlagAI)
		require.NoError(t, err)

		innerSchema := schema.Properties["inner"]
		data, err := json.Marshal(&innerSchema)
		require.NoError(t, err)

		var out map[string]any
		require.NoError(t, json.Unmarshal(data, &out))

		require.Equal(t, false, out["additionalProperties"],
			"nested object must also have additionalProperties:false")
	})
}

func TestBuildSchemaValidation(t *testing.T) {
	t.Parallel()

	t.Run("RequiredFieldEnforced", func(t *testing.T) {
		t.Parallel()

		schema := &lib.RawSchema{
			Type: lib.NewSchemaType("object"),
			Properties: map[string]lib.RawSchema{
				"name": {Type: lib.NewSchemaType("string")},
			},
			Required: []string{"name"},
		}

		resolved, err := lib.BuildSchema(schema)
		require.NoError(t, err)

		// Missing required field should fail.
		err = resolved.Validate(map[string]any{})
		require.Error(t, err)

		// Providing the required field should pass.
		err = resolved.Validate(map[string]any{"name": "alice"})
		require.NoError(t, err)
	})

	t.Run("PartialFlagRemovesRequired", func(t *testing.T) {
		t.Parallel()

		schema := &lib.RawSchema{
			Type: lib.NewSchemaType("object"),
			Properties: map[string]lib.RawSchema{
				"name": {Type: lib.NewSchemaType("string")},
			},
			Required: []string{"name"},
		}

		resolved, err := lib.BuildSchema(schema, lib.SchemaBuildFlagPartial)
		require.NoError(t, err)

		// With Partial flag, required constraint is dropped — missing field is OK.
		err = resolved.Validate(map[string]any{})
		require.NoError(t, err)

		// Providing the field is still accepted.
		err = resolved.Validate(map[string]any{"name": "alice"})
		require.NoError(t, err)
	})
}
