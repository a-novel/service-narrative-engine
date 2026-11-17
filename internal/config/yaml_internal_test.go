package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalYAML(t *testing.T) {
	t.Parallel()

	type window struct {
		InputTokens int `yaml:"inputTokens"`
	}

	type document struct {
		Windows map[string]window `yaml:"windows"`
	}

	testCases := []struct {
		name string

		data string

		expect    document
		expectErr bool
	}{
		{
			name: "Success",

			data: "windows:\n  daily:\n    inputTokens: 5\n",

			expect: document{Windows: map[string]window{"daily": {InputTokens: 5}}},
		},
		{
			// A misspelled key would read as its zero value, as a cap of 0 tokens.
			name: "Error/UnknownNestedKey",

			data: "windows:\n  daily:\n    inputToken: 5\n",

			expectErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var decoded document

			err := unmarshalYAML([]byte(testCase.data), &decoded)
			if testCase.expectErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, testCase.expect, decoded)
		})
	}
}
