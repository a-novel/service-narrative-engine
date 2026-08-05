package lib_test

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

//go:embed testdata/content-schema.yaml
var contentSchemaYAML []byte

func TestContentSchemaValidate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		value json.RawMessage

		expectErr bool
	}{
		{
			name:  "Success",
			value: json.RawMessage(`{"title":"The Answering Light","details":{"summary":"A reply beneath the sea."}}`),
		},
		{
			name:      "Error/MissingRequired",
			value:     json.RawMessage(`{"title":"The Answering Light"}`),
			expectErr: true,
		},
		{
			name:      "Error/NestedRequired",
			value:     json.RawMessage(`{"title":"The Answering Light","details":{}}`),
			expectErr: true,
		},
		{
			name:      "Error/WrongType",
			value:     json.RawMessage(`{"title":42,"details":{"summary":"A reply."}}`),
			expectErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			schema := lib.NewContentSchema(yamlToJSON(t, contentSchemaYAML), 1_024)

			instance, err := schema.Validate(testCase.value)

			if testCase.expectErr {
				require.Error(t, err)
				require.Nil(t, instance)
			} else {
				require.NoError(t, err)
				require.Equal(t, "The Answering Light", instance["title"])
			}
		})
	}
}

func TestContentSchemaLimitsAndPreparationErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		schema   json.RawMessage
		maxBytes int
		value    json.RawMessage

		expectSchemaErr bool
	}{
		{
			name:            "InvalidSchema",
			schema:          json.RawMessage(`{"$ref":"#/$defs/missing"}`),
			maxBytes:        1_024,
			value:           json.RawMessage(`{}`),
			expectSchemaErr: true,
		},
		{
			name:            "MalformedSchema",
			schema:          json.RawMessage(`{"type":`),
			maxBytes:        1_024,
			value:           json.RawMessage(`{}`),
			expectSchemaErr: true,
		},
		{
			name:     "DocumentTooLarge",
			schema:   json.RawMessage(`{"type":"object"}`),
			maxBytes: 1,
			value:    json.RawMessage(`{}`),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			schema := lib.NewContentSchema(testCase.schema, testCase.maxBytes)

			_, err := schema.Validate(testCase.value)

			require.Error(t, err)

			if testCase.expectSchemaErr {
				require.ErrorIs(t, err, lib.ErrContentSchemaInvalid)
			} else {
				require.NotErrorIs(t, err, lib.ErrContentSchemaInvalid)
			}
		})
	}
}

func TestContentSchemaRejectsInvalidDocuments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		value json.RawMessage
	}{
		{name: "Empty"},
		{name: "Malformed", value: json.RawMessage(`{`)},
		{name: "Null", value: json.RawMessage(`null`)},
		{name: "Array", value: json.RawMessage(`[]`)},
		{name: "InvalidUTF8", value: json.RawMessage{'{', '"', 0xff, '"', ':', '1', '}'}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			schema := lib.NewContentSchema(json.RawMessage(`{"type":"object"}`), 0)

			instance, err := schema.Validate(testCase.value)

			require.Error(t, err)
			require.Nil(t, instance)
		})
	}
}

func TestContentSchemaCopiesDefinition(t *testing.T) {
	t.Parallel()

	source := json.RawMessage(`{"type":"object"}`)
	schema := lib.NewContentSchema(source, 1_024)

	source[0] = 'x'

	instance, err := schema.Validate(json.RawMessage(`{}`))

	require.NoError(t, err)
	require.NotNil(t, instance)
}
