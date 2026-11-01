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

	const local = `&grpcf.LocalCredentialsProvider{}`

	testCases := []struct {
		name string

		restTimeoutRead string
		env             []string

		expect    string
		expectErr string
	}{
		{
			name:   "Success/Default",
			expect: "15s " + local + " " + local,
		},
		{
			name:            "Success",
			restTimeoutRead: "2s",
			expect:          "2s " + local + " " + local,
		},
		{
			name: "Success/ServiceTokens",
			env: []string{
				"TEST_SERVICE_JSON_KEYS_TOKEN_PATH=/tokens/json-keys",
				"TEST_SERVICE_GENAI_TOKEN_PATH=/tokens/genai",
			},
			expect: "15s" +
				` &grpcf.ServiceTokenCredentialsProvider{ServiceTokenPath:"/tokens/json-keys", TLS:false}` +
				` &grpcf.ServiceTokenCredentialsProvider{ServiceTokenPath:"/tokens/genai", TLS:false}`,
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
				"SERVICE_NARRATIVE_ENGINE_ENV_PREFIX=TEST_",
				"REST_TIMEOUT_READ=invalid",
				"TEST_REST_TIMEOUT_READ="+testCase.restTimeoutRead,
			)
			command.Env = append(command.Env, testCase.env...)

			output, err := command.CombinedOutput()
			if testCase.expectErr != "" {
				require.Error(t, err)
				require.Contains(t, string(output), testCase.expectErr)

				return
			}

			require.NoError(t, err, string(output))
			require.Equal(t, testCase.expect, strings.TrimSpace(string(output)))
		})
	}
}
