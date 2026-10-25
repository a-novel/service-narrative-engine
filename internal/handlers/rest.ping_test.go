package handlers_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/handlers"
)

func TestRestPing(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		expectStatus int
		expectBody   string
	}{
		{
			name:         "Success",
			expectStatus: http.StatusOK,
			expectBody:   "pong",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()

			handlers.NewRestPing().ServeHTTP(
				recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v0/ping", nil),
			)

			response := recorder.Result()
			require.Equal(t, testCase.expectStatus, response.StatusCode)

			body, err := io.ReadAll(response.Body)
			require.NoError(t, errors.Join(err, response.Body.Close()))
			require.Equal(t, testCase.expectBody, string(body))
		})
	}
}
