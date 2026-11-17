package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/servicetoken"
	"github.com/a-novel-kit/golib/servicetoken/servicetokentest"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

func TestNewRouter(t *testing.T) {
	t.Parallel()

	const (
		audience = "service-narrative-engine"
		caller   = "system:serviceaccount:agora:caller"
	)

	issuer := servicetokentest.NewIssuer(t)
	gated := servicetoken.Config{Issuer: issuer.URL, Audience: audience, Callers: []string{caller}}

	// The client connects lazily, so nothing listens at its address: the JSON Keys probe reports down.
	jsonKeysClient, err := servicejsonkeys.NewClient(
		"127.0.0.1:1", grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(jsonKeysClient.Close)

	// The test hands the router no database, no JSON Keys server and no genai host, so a request the
	// gate admits to the health route answers that the first two are down, and leaves genai out.
	bothDown := `{"api:jsonKeys":{"status":"down"},"client:postgres":{"status":"down"}}`

	testCases := []struct {
		name string

		serviceToken servicetoken.Config
		target       string
		token        string

		expectStatus int
		expectBody   string
		expectErr    error
	}{
		{
			// A caller's token passes the gate.
			name: "Success",

			serviceToken: gated,
			target:       "/v1/healthcheck",
			token:        issuer.Sign(t, issuer.Claims(caller, audience)),

			expectStatus: http.StatusServiceUnavailable,
			expectBody:   bothDown,
		},
		{
			// The liveness probe passes the gate without a token.
			name: "Success/Ping",

			serviceToken: gated,
			target:       "/v1/ping",

			expectStatus: http.StatusOK,
			expectBody:   "pong",
		},
		{
			// A local stack opts out, and the request reaches the health route.
			name: "Success/IssuerOptional",

			serviceToken: servicetoken.Config{IssuerOptional: true},
			target:       "/v1/healthcheck",

			expectStatus: http.StatusServiceUnavailable,
			expectBody:   bothDown,
		},
		{
			// The health route carries no auth of its own, so the refusal can only come from the gate.
			name: "Error/Unauthenticated",

			serviceToken: gated,
			target:       "/v1/healthcheck",

			expectStatus: http.StatusUnauthorized,
		},
		{
			// A deployment that forgets the issuer stops at startup rather than admit every caller.
			name: "Error/IssuerRequired",

			expectErr: servicetoken.ErrIssuerRequired,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.AppPresetDefault
			cfg.ServiceToken = testCase.serviceToken
			cfg.Dependencies.ServiceGenai = grpcf.ClientConfig{}

			router, err := newRouter(t.Context(), cfg, jsonKeysClient, unconfiguredGenai{})
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr != nil {
				return
			}

			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, testCase.target, nil)
			if testCase.token != "" {
				request.Header.Set(servicetoken.Header, testCase.token)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, request)

			require.Equal(t, testCase.expectStatus, w.Code)

			if testCase.expectBody != "" {
				require.Equal(t, testCase.expectBody, strings.TrimSpace(w.Body.String()))
			}
		})
	}
}
