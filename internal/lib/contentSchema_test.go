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

//go:embed testdata/content-schema-draft7.yaml
var contentSchemaDraft7YAML []byte

func TestContentSchemaValidateComplete(t *testing.T) {
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
			name:      "Error/WrongType",
			value:     json.RawMessage(`{"title":42,"details":{"summary":"A reply."}}`),
			expectErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			schema := lib.NewContentSchema(yamlToJSON(t, contentSchemaYAML), 1_024)

			instance, err := schema.ValidateComplete(testCase.value)

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

func TestContentSchemaValidatePartial(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		value json.RawMessage

		expectErr bool
	}{
		{
			name:  "Success/Empty",
			value: json.RawMessage(`{}`),
		},
		{
			name:  "Success/NestedRequiredOmitted",
			value: json.RawMessage(`{"details":{}}`),
		},
		{
			name:  "Success/OneOfPresenceRelaxed",
			value: json.RawMessage(`{"selection":{}}`),
		},
		{
			name:  "Success/ReferencedPresenceRelaxed",
			value: json.RawMessage(`{"referencedSelection":{}}`),
		},
		{
			name:  "Success/AnchoredPresenceRelaxed",
			value: json.RawMessage(`{"anchoredSelection":{}}`),
		},
		{
			name:  "Success/NestedResourcePresenceRelaxed",
			value: json.RawMessage(`{"nestedResourceSelection":{}}`),
		},
		{
			name:  "Success/LegacyDependencyRelaxed",
			value: json.RawMessage(`{"legacy":"present"}`),
		},
		{
			name:      "Error/WrongShape",
			value:     json.RawMessage(`{"details":"unknown"}`),
			expectErr: true,
		},
		{
			name:      "Error/ScalarOneOfPreserved",
			value:     json.RawMessage(`{"numericSelection":1}`),
			expectErr: true,
		},
		{
			name:      "Error/CyclicPresencePreserved",
			value:     json.RawMessage(`{"cyclicSelection":{}}`),
			expectErr: true,
		},
		{
			name:      "Error/PredicatePreserved",
			value:     json.RawMessage(`{"forbidden":"present"}`),
			expectErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			schema := lib.NewContentSchema(yamlToJSON(t, contentSchemaYAML), 1_024)

			instance, err := schema.ValidatePartial(testCase.value)

			if testCase.expectErr {
				require.Error(t, err)
				require.Nil(t, instance)
			} else {
				require.NoError(t, err)
				require.NotNil(t, instance)
			}
		})
	}
}

func TestContentSchemaValidatePartialDraft7References(t *testing.T) {
	t.Parallel()

	schema := lib.NewContentSchema(yamlToJSON(t, contentSchemaDraft7YAML), 1_024)

	instance, err := schema.ValidatePartial(json.RawMessage(`{"selection":{}}`))

	require.NoError(t, err)
	require.NotNil(t, instance)
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

			_, err := schema.ValidateComplete(testCase.value)

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

			instance, err := schema.ValidateComplete(testCase.value)

			require.Error(t, err)
			require.Nil(t, instance)
		})
	}
}

func TestContentSchemaJSONIsIndependent(t *testing.T) {
	t.Parallel()

	source := json.RawMessage(`{"type":"object"}`)
	expect := append(json.RawMessage(nil), source...)
	schema := lib.NewContentSchema(source, 1_024)

	source[0] = 'x'
	first := schema.JSON()
	first[0] = 'x'

	require.Equal(t, expect, schema.JSON())
}
