package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
)

func TestHealth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *http.Request

		skipPostgres bool

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil),

			expectResponse: map[string]any{
				"client:postgres": map[string]any{
					"status": handlers.RestHealthStatusUp,
				},
			},
			expectStatus: http.StatusOK,
		},
		{
			// Omitting postgres from the context makes the probe fail, so the entry reports
			// status=down. The exact match on expectResponse guards the public shape: it
			// fails if any extra field, such as a re-introduced "err", leaks into it.
			name: "Success/Degraded",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil),

			skipPostgres: true,

			expectResponse: map[string]any{
				"client:postgres": map[string]any{
					"status": handlers.RestHealthStatusDown,
				},
			},
			expectStatus: http.StatusOK,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			handler := handlers.NewRestHealth()
			w := httptest.NewRecorder()

			rCtx := testCase.request.Context()
			if !testCase.skipPostgres {
				var err error

				rCtx, err = postgres.NewContext(rCtx, configtest.PostgresPreset)
				require.NoError(t, err)
			}

			handler.ServeHTTP(w, testCase.request.WithContext(rCtx))

			res := w.Result()

			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				data, err := io.ReadAll(res.Body)
				require.NoError(t, errors.Join(err, res.Body.Close()))

				var jsonRes any
				require.NoError(t, json.Unmarshal(data, &jsonRes))
				require.Equal(t, testCase.expectResponse, jsonRes)
			}
		})
	}
}
