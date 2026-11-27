package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
	"github.com/a-novel/service-json-keys/v2/pkg/go/servicejsonkeystest"

	"github.com/a-novel-kit/golib/httpf"

	"github.com/a-novel/service-narrative-engine/v2/internal/config"
	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/config/env"
	"github.com/a-novel/service-narrative-engine/v2/internal/handlers"
)

func TestRestLimits(t *testing.T) {
	t.Parallel()

	otherUserID := uuid.MustParse("00000000-0000-0000-0000-0000000000a2")

	const (
		content = "/v1/instance/content"
		links   = "/v1/links"
	)

	// documentWrite is one write a test case sends, as a user, to a route.
	type documentWrite struct {
		userID uuid.UUID
		path   string

		expectStatus int
	}

	// writes returns count writes of a user to a route, which all pass.
	writes := func(count int, userID uuid.UUID, path string) []documentWrite {
		sent := make([]documentWrite, count)
		for index := range sent {
			sent[index] = documentWrite{userID: userID, path: path, expectStatus: http.StatusOK}
		}

		return sent
	}

	// The default rate lets 3 writes through each minute.
	perMinute := env.RestRateLimitDocumentWriteDefault.Requests

	testCases := []struct {
		name string

		// rateLimitsDisabled turns the rates off, as a test stack does.
		rateLimitsDisabled bool
		writes             []documentWrite
	}{
		{
			name: "Success",

			writes: writes(perMinute, ownerID, content),
		},
		{
			// Each user writes on a budget of their own.
			name: "Success/PerUser",

			writes: append(writes(perMinute, ownerID, content), writes(1, otherUserID, content)...),
		},
		{
			// Each route that writes a document counts on a budget of its own.
			name: "Success/PerRoute",

			writes: append(writes(perMinute, ownerID, content), writes(1, ownerID, links)...),
		},
		{
			name: "Success/RateLimitsDisabled",

			rateLimitsDisabled: true,
			writes:             writes(perMinute+1, ownerID, content),
		},
		{
			name: "Error/OneWriteTooMany",

			writes: append(
				writes(perMinute, ownerID, content),
				documentWrite{userID: ownerID, path: content, expectStatus: http.StatusTooManyRequests},
			),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// The case sends its writes at once, well inside the minute the rate counts over.
			limit := handlers.NewRestLimits(
				httpf.ServerConfig{RateLimitsEnabled: !testCase.rateLimitsDisabled},
				config.Limits{RateLimitDocumentWrite: env.RestRateLimitDocumentWriteDefault},
			)

			answer := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

			router := chi.NewRouter()
			router.With(limit.DocumentWrite).Put(content, answer)
			router.With(limit.DocumentWrite).Put(links, answer)

			for _, write := range testCase.writes {
				w := httptest.NewRecorder()
				request := httptest.NewRequestWithContext(t.Context(), http.MethodPut, write.path, nil)
				router.ServeHTTP(w, asUser(request, &write.userID))

				require.Equal(t, write.expectStatus, w.Code, write.path)

				// A refusal says when to try again.
				if write.expectStatus == http.StatusTooManyRequests {
					require.NotEmpty(t, w.Header().Get("Retry-After"))
				}
			}
		})
	}

	// The route that publishes a version, as the server mounts it: the publisher gate verifies the key of a
	// request, then the limit counts it. The gate takes a request that carries no token, as a local stack does.
	const purpose = servicejsonkeys.APIKeyPurposePublishNarrativeEnginePlugin

	publisher := servicejsonkeystest.NewAPIKey(t, purpose)
	otherPublisher := servicejsonkeystest.NewAPIKey(t, purpose)

	// forger signs with a key of its own under the publisher's ID, as a caller who only knows that ID does.
	forger := servicejsonkeystest.NewAPIKey(t, purpose)
	forger.ID = publisher.ID

	verifier, err := servicejsonkeys.NewAPIKeyVerifier(servicejsonkeystest.APIKeys{publisher, otherPublisher}, purpose)
	require.NoError(t, err)

	gate := servicejsonkeys.NewAPIKeyGate(
		verifier, servicejsonkeys.APIKeyGateConfig{TokenOptional: true}, configtest.Logger,
	)

	twoAMinute := httpf.Rate{Requests: 2, Window: time.Minute}
	fiveAQuarter := httpf.Rate{Requests: 5, Window: 15 * time.Minute}

	// publication is one version a test case publishes, with a request token key signed. A nil key sends none.
	type publication struct {
		key *servicejsonkeystest.APIKey

		expectStatus int
	}

	publishCases := []struct {
		name string

		// rateLimitsDisabled turns the rates off, as a test stack does.
		rateLimitsDisabled bool
		// burst and sustained are the short window and the long one. The case publishes at once.
		burst, sustained httpf.Rate
		publications     []publication
	}{
		{
			name: "Success/PluginVersion",

			burst: twoAMinute, sustained: fiveAQuarter,
			publications: []publication{
				{key: publisher, expectStatus: http.StatusCreated},
				{key: publisher, expectStatus: http.StatusCreated},
			},
		},
		{
			// Each API key publishes on a budget of its own.
			name: "Success/PluginVersion/PerKey",

			burst: twoAMinute, sustained: fiveAQuarter,
			publications: []publication{
				{key: publisher, expectStatus: http.StatusCreated},
				{key: publisher, expectStatus: http.StatusCreated},
				{key: otherPublisher, expectStatus: http.StatusCreated},
			},
		},
		{
			// A local stack publishes without a key, on the rate of every route alone.
			name: "Success/PluginVersion/NoKey",

			burst: twoAMinute, sustained: fiveAQuarter,
			publications: []publication{
				{expectStatus: http.StatusCreated},
				{expectStatus: http.StatusCreated},
				{expectStatus: http.StatusCreated},
			},
		},
		{
			// A token the gate refuses spends nothing of the budget of the key it names.
			name: "Success/PluginVersion/RefusedToken",

			burst: twoAMinute, sustained: fiveAQuarter,
			publications: []publication{
				{key: forger, expectStatus: http.StatusUnauthorized},
				{key: forger, expectStatus: http.StatusUnauthorized},
				{key: publisher, expectStatus: http.StatusCreated},
				{key: publisher, expectStatus: http.StatusCreated},
			},
		},
		{
			name: "Success/PluginVersion/RateLimitsDisabled",

			rateLimitsDisabled: true,
			burst:              twoAMinute, sustained: fiveAQuarter,
			publications: []publication{
				{key: publisher, expectStatus: http.StatusCreated},
				{key: publisher, expectStatus: http.StatusCreated},
				{key: publisher, expectStatus: http.StatusCreated},
			},
		},
		{
			name: "Error/PluginVersion/ThirdInTheShortWindow",

			burst: twoAMinute, sustained: fiveAQuarter,
			publications: []publication{
				{key: publisher, expectStatus: http.StatusCreated},
				{key: publisher, expectStatus: http.StatusCreated},
				{key: publisher, expectStatus: http.StatusTooManyRequests},
			},
		},
		{
			// The long window counts on its own: here it is the tighter of the two.
			name: "Error/PluginVersion/ThirdInTheLongWindow",

			burst:     httpf.Rate{Requests: 5, Window: time.Minute},
			sustained: httpf.Rate{Requests: 2, Window: 15 * time.Minute},
			publications: []publication{
				{key: publisher, expectStatus: http.StatusCreated},
				{key: publisher, expectStatus: http.StatusCreated},
				{key: publisher, expectStatus: http.StatusTooManyRequests},
			},
		},
	}

	for _, testCase := range publishCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			limit := handlers.NewRestLimits(
				httpf.ServerConfig{RateLimitsEnabled: !testCase.rateLimitsDisabled},
				config.Limits{
					RateLimitPluginVersionWriteBurst: testCase.burst,
					RateLimitPluginVersionWrite:      testCase.sustained,
				},
			)

			answer := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) }

			router := chi.NewRouter()
			router.With(gate, limit.PluginVersionWrite).Post("/v1/plugin/version", answer)

			for index, publication := range testCase.publications {
				body := []byte(`{"plugin": "scenes-composer"}`)

				w := httptest.NewRecorder()
				request := httptest.NewRequestWithContext(
					t.Context(), http.MethodPost, "/v1/plugin/version", bytes.NewReader(body),
				)

				if publication.key != nil {
					token := publication.key.Sign(
						t, servicejsonkeystest.TypRequestToken, publication.key.RequestClaims(body),
					)
					request.Header.Set("Authorization", "Bearer "+token)
				}

				router.ServeHTTP(w, request)

				require.Equal(t, publication.expectStatus, w.Code, "publication %d", index)

				// A refusal says when to try again.
				if publication.expectStatus == http.StatusTooManyRequests {
					require.NotEmpty(t, w.Header().Get("Retry-After"))
				}
			}
		})
	}
}
