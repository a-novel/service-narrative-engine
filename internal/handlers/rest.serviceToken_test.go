package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/servicetoken"
	"github.com/a-novel-kit/golib/servicetoken/servicetokentest"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
)

const (
	// serviceTokenAudience is the audience the service token tests answer to.
	serviceTokenAudience = "service-narrative-engine"
	// serviceTokenCaller is the one caller the service token tests allow.
	serviceTokenCaller = "system:serviceaccount:agora:studio"
	// serviceTokenStranger is a caller outside the allowed ones.
	serviceTokenStranger = "system:serviceaccount:agora:stranger"
)

func TestRestServiceToken(t *testing.T) {
	t.Parallel()

	issuer := servicetokentest.NewIssuer(t)
	tokenConfig := servicetoken.Config{
		Issuer:   issuer.URL,
		Audience: serviceTokenAudience,
		Callers:  []string{serviceTokenCaller},
	}

	testCases := []struct {
		name string

		tokenConfig servicetoken.Config
		path        string
		token       string

		expectStatus int
	}{
		{
			name: "Success",

			tokenConfig: tokenConfig,
			path:        "/v1/projects",
			token:       issuer.Sign(t, issuer.Claims(serviceTokenCaller, serviceTokenAudience)),

			expectStatus: http.StatusNoContent,
		},
		{
			name: "Success/Unconfigured",

			path: "/v1/projects",

			expectStatus: http.StatusNoContent,
		},
		{
			name: "Success/Ping",

			tokenConfig: tokenConfig,
			path:        "/v1/ping",

			expectStatus: http.StatusNoContent,
		},
		{
			name: "Error/UnknownCaller",

			tokenConfig: tokenConfig,
			path:        "/v1/projects",
			token:       issuer.Sign(t, issuer.Claims(serviceTokenStranger, serviceTokenAudience)),

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/Missing",

			tokenConfig: tokenConfig,
			path:        "/v1/projects",

			expectStatus: http.StatusUnauthorized,
		},
	}

	// The router answers 204, so a 401 proves the middleware refused the request before the router
	// read its client IP header or counted it against a rate limit.
	router := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, testCase.path, nil)
			if testCase.token != "" {
				request.Header.Set(servicetoken.Header, testCase.token)
			}

			w := httptest.NewRecorder()
			withServiceToken := handlers.RestServiceToken(testCase.tokenConfig, &http.Client{}, config.LoggerDev, "/v1/ping")
			withServiceToken(router).ServeHTTP(w, request)

			require.Equal(t, testCase.expectStatus, w.Code)
		})
	}
}
