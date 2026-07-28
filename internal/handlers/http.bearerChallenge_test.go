package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/handlers"
)

func TestBearerChallenge(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		status          int
		expectChallenge string
	}{
		{
			name:            "Unauthorized",
			status:          http.StatusUnauthorized,
			expectChallenge: `Bearer realm="narrative-engine", error="invalid_token"`,
		},
		{name: "Forbidden", status: http.StatusForbidden},
		{name: "Success", status: http.StatusOK},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", nil)
			handler := handlers.BearerChallenge(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(testCase.status)
			}))

			handler.ServeHTTP(w, request)

			require.Equal(t, testCase.status, w.Code)
			require.Equal(t, testCase.expectChallenge, w.Header().Get("WWW-Authenticate"))
		})
	}
}
