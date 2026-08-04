package lib_test

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

//go:embed testdata/engine-definition.yaml
var engineDefinitionYAML []byte

func TestSelectEngineStep(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		definition json.RawMessage
		key        string

		expectPrompt string
		expectErr    error
	}{
		{
			name:         "Success",
			definition:   yamlToJSON(t, engineDefinitionYAML),
			key:          "scenes",
			expectPrompt: "Build the scenes.",
		},
		{
			name:       "Error/InvalidDefinition",
			definition: json.RawMessage(`{`),
			key:        "scenes",
			expectErr:  lib.ErrEngineDefinitionInvalid,
		},
		{
			name:       "Error/UnknownStep",
			definition: yamlToJSON(t, engineDefinitionYAML),
			key:        "draft",
			expectErr:  lib.ErrEngineStepNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			step, err := lib.SelectEngineStep(testCase.definition, testCase.key)

			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr == nil {
				require.Equal(t, testCase.key, step.Key)
				require.Equal(t, testCase.expectPrompt, step.PromptTemplate)
			}
		})
	}
}
