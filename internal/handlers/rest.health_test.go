package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	servicegenai "github.com/a-novel/service-genai/pkg/go"
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
		// withoutGenai runs the service with no genai host, which leaves genai out of the report.
		withoutGenai bool
		genaiErr     error

		expectStatus   int
		expectPostgres string
		expectJSONKeys string
		expectGenai    string
	}{
		{
			name:           "Success",
			setup:          probeHealthy,
			expectStatus:   http.StatusOK,
			expectPostgres: "up",
			expectJSONKeys: "up",
			expectGenai:    "up",
		},
		{
			name:           "Success/WithoutGenai",
			setup:          probeHealthy,
			withoutGenai:   true,
			expectStatus:   http.StatusOK,
			expectPostgres: "up",
			expectJSONKeys: "up",
		},
		{
			name:           "Error/GenaiDown",
			setup:          probeHealthy,
			genaiErr:       errors.New("unavailable"),
			expectStatus:   http.StatusServiceUnavailable,
			expectPostgres: "up",
			expectJSONKeys: "up",
			expectGenai:    "down",
		},
		{
			name:           "Error/JSONKeysDown",
			setup:          probeHealthy,
			jsonKeysErr:    errors.New("unavailable"),
			expectStatus:   http.StatusServiceUnavailable,
			expectPostgres: "up",
			expectJSONKeys: "down",
			expectGenai:    "up",
		},
		{
			name:           "Error/MissingPostgres",
			setup:          probeWithoutPostgres,
			expectStatus:   http.StatusServiceUnavailable,
			expectPostgres: "down",
			expectJSONKeys: "up",
			expectGenai:    "up",
		},
		{
			name:           "Error/ClosedPostgres",
			setup:          probeClosedPostgres,
			expectStatus:   http.StatusServiceUnavailable,
			expectPostgres: "down",
			expectJSONKeys: "up",
			expectGenai:    "up",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := newProbeContext(t, testCase.setup)
			recorder := httptest.NewRecorder()

			jsonKeys := handlersmocks.NewMockRestHealthAPIJSONKeys(t)
			jsonKeys.EXPECT().
				Status(mock.Anything, &servicejsonkeys.StatusRequest{}).
				Return(&servicejsonkeys.StatusResponse{}, testCase.jsonKeysErr)

			// A nil genai keeps its nil interface: a typed nil would read as configured.
			var genai handlers.RestHealthAPIGenai

			if !testCase.withoutGenai {
				genaiMock := handlersmocks.NewMockRestHealthAPIGenai(t)
				genaiMock.EXPECT().
					Status(mock.Anything, &servicegenai.StatusRequest{}).
					Return(&servicegenai.StatusResponse{}, testCase.genaiErr)

				genai = genaiMock
			}

			handlers.NewRestHealth(jsonKeys, genai).ServeHTTP(
				recorder, httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/healthcheck", nil),
			)

			// Exact matching keeps raw dependency errors out of the public report.
			expectReport := map[string]any{
				"client:postgres": map[string]any{"status": testCase.expectPostgres},
				"api:jsonKeys":    map[string]any{"status": testCase.expectJSONKeys},
			}
			if !testCase.withoutGenai {
				expectReport["api:genai"] = map[string]any{"status": testCase.expectGenai}
			}

			response := recorder.Result()
			require.Contains(t, response.Header.Get("Content-Type"), "application/json")
			requireResponse(t, response, testCase.expectStatus, expectReport)
		})
	}
}
