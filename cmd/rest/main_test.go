package main

import (
	"cmp"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
	"github.com/a-novel/service-json-keys/v2/pkg/go/servicejsonkeystest"

	"github.com/a-novel-kit/golib/servicetoken"
	"github.com/a-novel-kit/golib/servicetoken/servicetokentest"

	"github.com/a-novel/service-narrative-engine/v2/internal/config"
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

	// The test hands the router no database and no JSON Keys server, so a request the gate admits to
	// the health route answers that both are down.
	bothDown := `{"api:jsonKeys":{"status":"down"},"client:postgres":{"status":"down"}}`

	// The publish route, and a version the registry refuses whole. A request the publisher gate admits
	// answers that refusal, since the test hands the router no database to store a version in.
	const (
		publish = "/v1/plugin/version"
		version = `{}`
		purpose = servicejsonkeys.APIKeyPurposePublishNarrativeEnginePlugin
	)

	open := servicetoken.Config{IssuerOptional: true}
	serviceToken := issuer.Sign(t, issuer.Claims(caller, audience))

	// The API keys JSON Keys serves: the publisher's, one it deactivated, and one made for another purpose.
	// It serves no unknown key.
	key := servicejsonkeystest.NewAPIKey(t, purpose)
	unknownKey := servicejsonkeystest.NewAPIKey(t, purpose)
	deactivatedKey := servicejsonkeystest.NewAPIKey(t, purpose)
	deactivatedKey.Deactivated = true
	otherPurposeKey := servicejsonkeystest.NewAPIKey(t, "withdraw-narrative-engine-plugin")

	publisherKeys := servicejsonkeystest.WithAPIKeys(jsonKeysClient, key, deactivatedKey, otherPurposeKey)

	// bearer returns the Authorization header of a request whose token key signed for claims.
	bearer := func(key *servicejsonkeystest.APIKey, claims any) string {
		return "Bearer " + key.Sign(t, servicejsonkeystest.TypRequestToken, claims)
	}

	now := time.Now()

	// The claims of a request token whose five minutes ended an hour ago.
	expired := key.RequestClaims([]byte(version))
	expired["iat"] = now.Add(-time.Hour).Unix()
	expired["exp"] = now.Add(-time.Hour + servicejsonkeys.APIKeyRequestLifetime).Unix()

	// The claims of a user's access token, which JSON Keys signs with a key it serves as no API key.
	accessClaims := map[string]any{
		"userID": "00000000-0000-0000-0000-0000000000a1", "roles": []string{"auth:superadmin"},
		"iat": now.Unix(), "exp": now.Add(15 * time.Minute).Unix(),
	}

	testCases := []struct {
		name string

		serviceToken servicetoken.Config
		// maxItemSize caps the body of a route that writes one item, in bytes. The default cap stays when
		// it is zero.
		maxItemSize int64

		// method is the method of the request, GET when empty.
		method string
		target string
		body   string
		token  string
		// authorization is the Authorization header of the request, where a publisher sends its request
		// token.
		authorization string
		// requests is how many times the case sends its request, once when zero. The case checks the last
		// answer.
		requests int

		expectStatus int
		expectBody   string
		expectErr    error
	}{
		{
			// A caller's token passes the gate.
			name: "Success",

			serviceToken: gated,
			target:       "/v1/healthcheck",
			token:        serviceToken,

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
			// The gate wraps the whole router: an unknown path is refused before the router answers 404,
			// so a gate mounted inside /v1 fails this case.
			name: "Error/UnknownPath",

			serviceToken: gated,
			target:       "/nowhere",

			expectStatus: http.StatusUnauthorized,
		},
		{
			// A deployment that forgets the issuer stops at startup rather than admit every caller.
			name: "Error/IssuerRequired",

			expectErr: servicetoken.ErrIssuerRequired,
		},
		{
			// A request the publisher's API key signed for its body reaches the registry.
			name: "Success/Publisher",

			serviceToken:  gated,
			method:        http.MethodPost,
			target:        publish,
			body:          version,
			token:         serviceToken,
			authorization: bearer(key, key.RequestClaims([]byte(version))),

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			// A stack that runs without service tokens publishes without a key.
			name: "Success/Publisher/LocalStack",

			serviceToken: open,
			method:       http.MethodPost,
			target:       publish,
			body:         version,

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			// Each refused publication below carries a caller's token, so its refusal comes from the
			// publisher gate.
			name: "Error/Publisher/NoToken",

			serviceToken: gated,
			method:       http.MethodPost,
			target:       publish,
			body:         version,
			token:        serviceToken,

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/Publisher/UnknownKey",

			serviceToken:  gated,
			method:        http.MethodPost,
			target:        publish,
			body:          version,
			token:         serviceToken,
			authorization: bearer(unknownKey, unknownKey.RequestClaims([]byte(version))),

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/Publisher/DeactivatedKey",

			serviceToken:  gated,
			method:        http.MethodPost,
			target:        publish,
			body:          version,
			token:         serviceToken,
			authorization: bearer(deactivatedKey, deactivatedKey.RequestClaims([]byte(version))),

			expectStatus: http.StatusUnauthorized,
		},
		{
			// A key made for another purpose signs for that purpose alone.
			name: "Error/Publisher/OtherPurpose",

			serviceToken:  gated,
			method:        http.MethodPost,
			target:        publish,
			body:          version,
			token:         serviceToken,
			authorization: bearer(otherPurposeKey, otherPurposeKey.RequestClaims([]byte(version))),

			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Error/Publisher/Expired",

			serviceToken:  gated,
			method:        http.MethodPost,
			target:        publish,
			body:          version,
			token:         serviceToken,
			authorization: bearer(key, expired),

			expectStatus: http.StatusUnauthorized,
		},
		{
			// The token was signed for another body than the one the request carries.
			name: "Error/Publisher/OtherBody",

			serviceToken:  gated,
			method:        http.MethodPost,
			target:        publish,
			body:          version,
			token:         serviceToken,
			authorization: bearer(key, key.RequestClaims([]byte(`{"plugin": "other"}`))),

			expectStatus: http.StatusUnauthorized,
		},
		{
			// A user's access token cannot publish, whatever its roles.
			name: "Error/Publisher/AccessToken",

			serviceToken:  gated,
			method:        http.MethodPost,
			target:        publish,
			body:          version,
			token:         serviceToken,
			authorization: bearer(unknownKey, accessClaims),

			expectStatus: http.StatusUnauthorized,
		},
		{
			// The service token gate wraps the publish route too: a signed request still needs a caller's
			// token.
			name: "Error/Publisher/NoServiceToken",

			serviceToken:  gated,
			method:        http.MethodPost,
			target:        publish,
			body:          version,
			authorization: bearer(key, key.RequestClaims([]byte(version))),

			expectStatus: http.StatusUnauthorized,
		},
		{
			// The route reads its body under the cap of a route that writes one item.
			name: "Error/Publisher/BodyTooLarge",

			serviceToken:  gated,
			maxItemSize:   1,
			method:        http.MethodPost,
			target:        publish,
			body:          version,
			token:         serviceToken,
			authorization: bearer(key, key.RequestClaims([]byte(version))),

			expectStatus: http.StatusRequestEntityTooLarge,
		},
		{
			// The limit on the versions of a key comes after the publisher gate, which verifies that key: the
			// third version a key publishes within a minute is refused, whatever the first two answered.
			name: "Error/Publisher/TooManyVersions",

			serviceToken:  gated,
			method:        http.MethodPost,
			target:        publish,
			body:          version,
			token:         serviceToken,
			authorization: bearer(key, key.RequestClaims([]byte(version))),
			requests:      3,

			expectStatus: http.StatusTooManyRequests,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.AppPresetDefault
			cfg.ServiceToken = testCase.serviceToken
			cfg.Limits.MaxItemRequestSize = cmp.Or(testCase.maxItemSize, cfg.Limits.MaxItemRequestSize)

			router, err := newRouter(cfg, publisherKeys)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr != nil {
				return
			}

			var w *httptest.ResponseRecorder

			for range max(testCase.requests, 1) {
				request := httptest.NewRequestWithContext(
					t.Context(), cmp.Or(testCase.method, http.MethodGet), testCase.target,
					strings.NewReader(testCase.body),
				)
				if testCase.token != "" {
					request.Header.Set(servicetoken.Header, testCase.token)
				}

				if testCase.authorization != "" {
					request.Header.Set("Authorization", testCase.authorization)
				}

				w = httptest.NewRecorder()
				router.ServeHTTP(w, request)
			}

			require.Equal(t, testCase.expectStatus, w.Code)

			if testCase.expectBody != "" {
				require.Equal(t, testCase.expectBody, strings.TrimSpace(w.Body.String()))
			}
		})
	}
}
