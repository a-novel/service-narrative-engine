package config_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppPresetDefault(t *testing.T) {
	t.Parallel()

	// The preset resolves at package initialization, so each case reads it in a process of its own,
	// from one build of the app.
	app := filepath.Join(t.TempDir(), "app")

	//nolint:gosec // The output path is the test's own temporary directory.
	build := exec.CommandContext(t.Context(), "go", "build", "-o", app, "./testdata/app")

	build.Env = append(os.Environ(), "GOWORK=off")

	output, err := build.CombinedOutput()
	require.NoError(t, err, string(output))

	testCases := []struct {
		name string

		env []string

		expect string
	}{
		{
			// The unprefixed variable would fail to parse: reading the prefixed one proves the prefix
			// applies. Genai keeps no host, so the service runs without it.
			name:   "Success",
			env:    []string{"TEST_REST_TIMEOUT_READ=2s"},
			expect: "2s {Host:localhost Port:8080 TLS:false TokenPath:} {Host: Port:8080 TLS:false TokenPath:}",
		},
		{
			name: "Success/Dependencies",
			env: []string{
				"TEST_SERVICE_JSON_KEYS_HOST=json-keys",
				"TEST_SERVICE_JSON_KEYS_PORT=9090",
				"TEST_SERVICE_JSON_KEYS_TLS=true",
				"TEST_SERVICE_JSON_KEYS_TOKEN_PATH=/tokens/json-keys",
				"TEST_SERVICE_GENAI_HOST=genai",
				"TEST_SERVICE_GENAI_PORT=9091",
				"TEST_SERVICE_GENAI_TLS=true",
				"TEST_SERVICE_GENAI_TOKEN_PATH=/tokens/genai",
			},
			expect: "15s" +
				" {Host:json-keys Port:9090 TLS:true TokenPath:/tokens/json-keys}" +
				" {Host:genai Port:9091 TLS:true TokenPath:/tokens/genai}",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			command := exec.CommandContext(t.Context(), app)

			command.Env = append(os.Environ(), "SERVICE_NARRATIVE_ENGINE_ENV_PREFIX=TEST_", "REST_TIMEOUT_READ=invalid")
			command.Env = append(command.Env, testCase.env...)

			output, err := command.CombinedOutput()
			require.NoError(t, err, string(output))
			require.Equal(t, testCase.expect, strings.TrimSpace(string(output)))
		})
	}
}
