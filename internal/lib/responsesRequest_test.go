package lib_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestEncodeResponsesJSONSchemaRequest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *lib.ResponsesJSONSchemaRequest

		expectErr bool
	}{
		{
			name: "Success",
			request: &lib.ResponsesJSONSchemaRequest{
				Model:        "gpt-test",
				Reasoning:    "medium",
				Instructions: "Complete the target.",
				Input:        `{"target":"idea"}`,
				SchemaName:   "target_output",
				OutputSchema: json.RawMessage(`{"type":"object"}`),
			},
		},
		{
			name:      "Error/NilRequest",
			expectErr: true,
		},
		{
			name: "Error/MalformedSchema",
			request: &lib.ResponsesJSONSchemaRequest{
				OutputSchema: json.RawMessage(`{"type":`),
			},
			expectErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result, err := lib.EncodeResponsesJSONSchemaRequest(testCase.request)
			if testCase.expectErr {
				require.Error(t, err)
				require.Nil(t, result)

				return
			}

			require.NoError(t, err)

			var payload struct {
				Model        string `json:"model"`
				Instructions string `json:"instructions"`
				Input        string `json:"input"`
				Reasoning    struct {
					Effort string `json:"effort"`
				} `json:"reasoning"`
				Text struct {
					Format struct {
						Type   string         `json:"type"`
						Name   string         `json:"name"`
						Schema map[string]any `json:"schema"`
						Strict bool           `json:"strict"`
					} `json:"format"`
				} `json:"text"`
			}

			err = json.Unmarshal(result, &payload)
			require.NoError(t, err)
			require.Equal(t, testCase.request.Model, payload.Model)
			require.Equal(t, testCase.request.Reasoning, payload.Reasoning.Effort)
			require.Equal(t, testCase.request.Instructions, payload.Instructions)
			require.Equal(t, testCase.request.Input, payload.Input)
			require.Equal(t, "json_schema", payload.Text.Format.Type)
			require.Equal(t, testCase.request.SchemaName, payload.Text.Format.Name)
			require.Equal(t, map[string]any{"type": "object"}, payload.Text.Format.Schema)
			require.True(t, payload.Text.Format.Strict)
		})
	}
}
