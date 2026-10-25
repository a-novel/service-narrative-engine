package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestHealth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		setup       probeSetup
		jsonKeysErr error

		expectStatus   int
		expectHealth   string
		expectJsonKeys string
	}{
		{
			name:           "Success",
			setup:          probeHealthy,
			expectStatus:   http.StatusOK,
			expectHealth:   handlers.RestHealthStatusUp,
			expectJsonKeys: handlers.RestHealthStatusUp,
		},
		{
			name:           "Error/JsonKeysDown",
			setup:          probeHealthy,
			jsonKeysErr:    errors.New("unavailable"),
			expectStatus:   http.StatusServiceUnavailable,
			expectHealth:   handlers.RestHealthStatusUp,
			expectJsonKeys: handlers.RestHealthStatusDown,
		},
		{
			name:           "Error/MissingPostgres",
			setup:          probeWithoutPostgres,
			expectStatus:   http.StatusServiceUnavailable,
			expectHealth:   handlers.RestHealthStatusDown,
			expectJsonKeys: handlers.RestHealthStatusUp,
		},
		{
			name:           "Error/ClosedPostgres",
			setup:          probeClosedPostgres,
			expectStatus:   http.StatusServiceUnavailable,
			expectHealth:   handlers.RestHealthStatusDown,
			expectJsonKeys: handlers.RestHealthStatusUp,
		},
		{
			name:           "Error/TransactionContext",
			setup:          probeInTransaction,
			expectStatus:   http.StatusServiceUnavailable,
			expectHealth:   handlers.RestHealthStatusDown,
			expectJsonKeys: handlers.RestHealthStatusUp,
		},
		{
			name:           "Error/CancelledProbe",
			setup:          probeCancelled,
			expectStatus:   http.StatusServiceUnavailable,
			expectHealth:   handlers.RestHealthStatusDown,
			expectJsonKeys: handlers.RestHealthStatusUp,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := newProbeContext(t, testCase.setup)
			recorder := httptest.NewRecorder()

			jsonKeys := handlersmocks.NewMockRestHealthApiJsonKeys(t)
			jsonKeys.EXPECT().
				Status(mock.Anything, &servicejsonkeys.StatusRequest{}).
				Return(&servicejsonkeys.StatusResponse{}, testCase.jsonKeysErr)

			handlers.NewRestHealth(jsonKeys).ServeHTTP(
				recorder, httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/healthcheck", nil),
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
				"api:jsonKeys":    map[string]any{"status": testCase.expectJsonKeys},
			}, report)
		})
	}
}
