package lib_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestExtractResponsesOutputText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		output json.RawMessage

		expect    string
		expectErr error
	}{
		{
			name:   "Success/TopLevel",
			output: json.RawMessage(`{"output_text":"complete"}`),
			expect: "complete",
		},
		{
			name: "Success/Nested",
			output: json.RawMessage(
				`{"output":[{"content":[{"type":"output_text","text":"com"}]},` +
					`{"content":[{"type":"output_text","text":"plete"}]}]}`,
			),
			expect: "complete",
		},
		{
			name: "Error/RefusalPrecedesText",
			output: json.RawMessage(
				`{"output_text":"ignored","output":[{"content":[` +
					`{"type":"refusal","refusal":"declined"}]}]}`,
			),
			expectErr: lib.ErrResponsesRefused,
		},
		{
			name:      "Error/Empty",
			expectErr: lib.ErrResponsesOutputEmpty,
		},
		{
			name:      "Error/Malformed",
			output:    json.RawMessage(`{`),
			expectErr: lib.ErrResponsesOutputMalformed,
		},
		{
			name:      "Error/TextMissing",
			output:    json.RawMessage(`{}`),
			expectErr: lib.ErrResponsesOutputTextMissing,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result, err := lib.ExtractResponsesOutputText(testCase.output)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
