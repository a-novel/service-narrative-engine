package prompts_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config/prompts"
)

func TestLookup(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		kind     string
		language string

		expectErr error
	}{
		{
			name:     "Success",
			kind:     "suggest",
			language: "fr",
		},
		{
			name:      "Error/UnknownKind",
			kind:      "rewrite",
			language:  "fr",
			expectErr: prompts.ErrUnknownPrompt,
		},
		{
			name:      "Error/UnknownLanguage",
			kind:      "suggest",
			language:  "de",
			expectErr: prompts.ErrUnknownPrompt,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			set, err := prompts.Lookup(testCase.kind, testCase.language)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr != nil {
				return
			}

			for _, name := range []string{"developer", "rules", "frame", "scene", "task." + testCase.kind, "user"} {
				require.NotNil(t, set.Lookup(name), name)
			}
		})
	}
}
