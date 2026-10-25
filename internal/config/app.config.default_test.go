package config_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppPresetDefault(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		restTimeoutRead string

		expectRead string
		expectErr  string
	}{
		{
			name:       "Success/Default",
			expectRead: "15s",
		},
		{
			name:            "Success",
			restTimeoutRead: "2s",
			expectRead:      "2s",
		},
		{
			name:            "Error/InvalidRestTimeoutRead",
			restTimeoutRead: "invalid",
			expectErr:       `time: invalid duration "invalid"`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// The preset resolves at package initialization, so each case reads it in a process of its
			// own. The unprefixed variable proves the prefix is applied.
			command := exec.CommandContext(t.Context(), "go", "run", "./testdata/app")

			command.Env = append(os.Environ(),
				"GOWORK=off",
				"SERVICE_TEMPLATE_ENV_PREFIX=TEST_",
				"REST_TIMEOUT_READ=invalid",
				"TEST_REST_TIMEOUT_READ="+testCase.restTimeoutRead,
			)

			output, err := command.CombinedOutput()
			if testCase.expectErr != "" {
				require.Error(t, err)
				require.Contains(t, string(output), testCase.expectErr)

				return
			}

			require.NoError(t, err, string(output))
			require.Equal(t, testCase.expectRead, strings.TrimSpace(string(output)))
		})
	}
}
