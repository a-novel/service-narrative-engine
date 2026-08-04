package lib_test

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

//go:embed testdata/responses-schema.yaml
var responsesSchemaYAML []byte

func TestProjectResponsesJSONSchema(t *testing.T) {
	t.Parallel()

	source := yamlToJSON(t, responsesSchemaYAML)
	original := bytes.Clone(source)

	result, err := lib.ProjectResponsesJSONSchema(source)
	require.NoError(t, err)
	require.Equal(t, original, source)

	var schema map[string]any

	err = json.Unmarshal(result, &schema)
	require.NoError(t, err)
	require.NotContains(t, schema, "$schema")
	require.NotContains(t, schema, "title")
	require.Equal(t, false, schema["additionalProperties"])
	require.Equal(t, []any{"count", "kind", "nested", "tags"}, schema["required"])

	properties := schema["properties"].(map[string]any)
	kind := properties["kind"].(map[string]any)
	require.Equal(t, []any{"scene"}, kind["enum"])
	require.NotContains(t, kind, "const")
	require.NotContains(t, kind, "minLength")

	tags := properties["tags"].(map[string]any)
	require.EqualValues(t, 1, tags["minItems"])

	nested := properties["nested"].(map[string]any)
	alternatives := nested["anyOf"].([]any)
	nestedObject := alternatives[0].(map[string]any)
	require.Equal(t, false, nestedObject["additionalProperties"])
	require.Equal(
		t,
		[]any{"$schema", "const", "label", "minLength"},
		nestedObject["required"],
	)

	nestedProperties := nestedObject["properties"].(map[string]any)
	require.Contains(t, nestedProperties, "$schema")
	require.Contains(t, nestedProperties, "const")
	require.Contains(t, nestedProperties, "minLength")
	require.NotContains(t, nestedProperties["label"].(map[string]any), "maxLength")
}

func TestProjectResponsesJSONSchemaErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		source json.RawMessage

		expectError error
	}{
		{
			name:   "Malformed",
			source: json.RawMessage(`{"type":`),
		},
		{
			name:   "Null",
			source: json.RawMessage(`null`),
		},
		{
			name: "UnsupportedKeyword",
			source: json.RawMessage(
				`{"type":"object","unevaluatedProperties":false}`,
			),
			expectError: lib.ErrResponsesSchemaUnsupported,
		},
		{
			name: "ConstEnumConflict",
			source: json.RawMessage(
				`{"type":"string","const":"scene","enum":["chapter"]}`,
			),
			expectError: lib.ErrResponsesSchemaConflict,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result, err := lib.ProjectResponsesJSONSchema(testCase.source)
			require.Error(t, err)
			require.Nil(t, result)

			if testCase.expectError != nil {
				require.ErrorIs(t, err, testCase.expectError)
			}
		})
	}
}
