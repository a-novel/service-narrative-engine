package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-template/internal/handlers"
)

func TestRestHealth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		setup probeSetup

		expectStatus int
		expectHealth string
	}{
		{
			name:         "Success",
			setup:        probeHealthy,
			expectStatus: http.StatusOK,
			expectHealth: handlers.RestHealthStatusUp,
		},
		{
			name:         "Error/MissingPostgres",
			setup:        probeWithoutPostgres,
			expectStatus: http.StatusServiceUnavailable,
			expectHealth: handlers.RestHealthStatusDown,
		},
		{
			name:         "Error/ClosedPostgres",
			setup:        probeClosedPostgres,
			expectStatus: http.StatusServiceUnavailable,
			expectHealth: handlers.RestHealthStatusDown,
		},
		{
			name:         "Error/TransactionContext",
			setup:        probeInTransaction,
			expectStatus: http.StatusServiceUnavailable,
			expectHealth: handlers.RestHealthStatusDown,
		},
		{
			name:         "Error/CancelledProbe",
			setup:        probeCancelled,
			expectStatus: http.StatusServiceUnavailable,
			expectHealth: handlers.RestHealthStatusDown,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := newProbeContext(t, testCase.setup)
			recorder := httptest.NewRecorder()

			handlers.NewRestHealth().ServeHTTP(
				recorder, httptest.NewRequestWithContext(ctx, http.MethodGet, "/v0/healthcheck", nil),
			)

			response := recorder.Result()
			require.Equal(t, testCase.expectStatus, response.StatusCode)
			require.Contains(t, response.Header.Get("Content-Type"), "application/json")

			body, err := io.ReadAll(response.Body)
			require.NoError(t, errors.Join(err, response.Body.Close()))

			var report any
			require.NoError(t, json.Unmarshal(body, &report))
			// Exact matching keeps raw dependency errors out of the public report.
			require.Equal(t, map[string]any{
				"client:postgres": map[string]any{"status": testCase.expectHealth},
			}, report)
		})
	}
}
