package handlers_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
	"github.com/a-novel/service-json-keys/v2/pkg/go/servicejsonkeystest"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	"github.com/a-novel/service-narrative-engine/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/v2/internal/handlers/mocks"
)

func TestRestPluginVersionPublish(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	const (
		schema   = `{"type": "object", "default": {"beats": []}}`
		exports  = `{"scenes": {"name": "exports.scenes", "select": "$.beats[*]", "id": "/id", "label": "/title"}}`
		messages = `{"exports": {"scenes": "Scenes"}}`
		ui       = "export default { mount() {} };"
		migrate  = "export default (content) => ({ migrated: content });"

		english = `"en": {"title": "Scenes Composer", "description": "Lays a story out.", "messages": ` + messages + `}`
		french  = `"fr": {"title": "Compositeur", "description": "Découpe une histoire.", "messages": ` + messages + `}`

		// The members every version carries, before its migration and its languages.
		version = `"plugin": "scenes-composer", "version": "1.10.0", "schema": ` + schema + `, "exports": ` + exports +
			`, "ui": "` + ui + `"`

		proof   = "header.claims.signature"
		purpose = servicejsonkeys.APIKeyPurposePublishNarrativeEnginePlugin
	)

	// The route's gate, as a local stack mounts it: it authenticates a request that carries a request token,
	// and takes one that carries none.
	key := servicejsonkeystest.NewAPIKey(t, purpose)

	verifier, err := servicejsonkeys.NewAPIKeyVerifier(servicejsonkeystest.APIKeys{key}, purpose)
	require.NoError(t, err)

	gate := servicejsonkeys.NewAPIKeyGate(
		verifier, servicejsonkeys.APIKeyGateConfig{TokenOptional: true}, configtest.Logger,
	)

	// published returns the version the handler asks the service for, changed by change.
	published := func(change func(request *core.PluginVersionPublishRequest)) *core.PluginVersionPublishRequest {
		request := &core.PluginVersionPublishRequest{
			Plugin: "scenes-composer", Version: "1.10.0", Schema: []byte(schema), Exports: []byte(exports),
			UI: []byte(ui), Migration: []byte(migrate),
			Languages: map[string]core.PluginVersionPublishLanguage{
				"en": {Title: "Scenes Composer", Description: "Lays a story out.", Messages: []byte(messages)},
				"fr": {Title: "Compositeur", Description: "Découpe une histoire.", Messages: []byte(messages)},
			},
		}
		change(request)

		return request
	}

	// A request whose English lacks its title, as the service's validator reports it: under the name of the
	// request's type, which the refusal leaves out.
	type refusedLanguage struct {
		Title string `validate:"required"`
	}

	type refusedRequest struct {
		Languages map[string]refusedLanguage `validate:"dive"`
	}

	untitled := refusedRequest{Languages: map[string]refusedLanguage{"en": {}}}

	whole := `{` + version + `, "migration": "` + migrate + `", "languages": {` + english + `, ` + french + `}}`
	proven := `{` + version + `, "migration": "` + migrate + `", "languages": {` + english + `, ` + french +
		`}, "proof": "` + proof + `"}`

	testCases := []struct {
		name string

		body string
		// signed sends the request with a request token the test's API key signed for the body.
		signed bool

		// serviceRequest is the version the handler asks the service for. The service is not called when it is
		// nil.
		serviceRequest *core.PluginVersionPublishRequest
		serviceErr     error

		expectStatus   int
		expectLocation string
		expectResponse any
	}{
		{
			name: "Success",

			body: whole,

			serviceRequest: published(func(*core.PluginVersionPublishRequest) {}),

			expectStatus:   http.StatusCreated,
			expectLocation: "/v1/plugin/version?plugin=scenes-composer&version=1.10.0",
		},
		{
			// The service receives the proof of the body, and the ID of the key that authenticated the request.
			name: "Success/APIKey",

			body:   proven,
			signed: true,

			serviceRequest: published(func(request *core.PluginVersionPublishRequest) {
				request.Proof, request.APIKeyID = proof, key.ID
			}),

			expectStatus:   http.StatusCreated,
			expectLocation: "/v1/plugin/version?plugin=scenes-composer&version=1.10.0",
		},
		{
			// A version without a migration carries no migration member.
			name: "Success/NoMigration",

			body: `{` + version + `, "languages": {` + english + `, ` + french + `}}`,

			serviceRequest: published(func(request *core.PluginVersionPublishRequest) { request.Migration = nil }),

			expectStatus:   http.StatusCreated,
			expectLocation: "/v1/plugin/version?plugin=scenes-composer&version=1.10.0",
		},
		{
			// An empty migration is not an absent one: the service refuses it.
			name: "Success/EmptyMigration",

			body: `{` + version + `, "migration": "", "languages": {` + english + `, ` + french + `}}`,

			serviceRequest: published(func(request *core.PluginVersionPublishRequest) { request.Migration = []byte{} }),

			expectStatus:   http.StatusCreated,
			expectLocation: "/v1/plugin/version?plugin=scenes-composer&version=1.10.0",
		},
		{
			// The service decides what a version lacks: the handler hands over what the body holds.
			name: "Success/EmptyBody",

			body: `{}`,

			serviceRequest: &core.PluginVersionPublishRequest{
				UI: []byte{}, Languages: map[string]core.PluginVersionPublishLanguage{},
			},
			serviceErr: core.ErrInvalidRequest,

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/MalformedRequest",

			body: `{"plugin": "scenes-composer",`,

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/ModuleNotAString",

			body: `{` + version + `, "migration": {"source": "export default 1;"}}`,

			expectStatus: http.StatusBadRequest,
		},
		{
			// A version never changes once published, so a member the registry does not know is refused: stored
			// without it, the version would lack it for good. Nothing reaches the service.
			name: "Error/UnknownMember",

			body: `{` + version + `, "languages": {` + english + `}, "links": {"scenes": []}}`,

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/UnknownMemberOfALanguage",

			body: `{` + version + `, "languages": {"en": {"title": "Scenes Composer", "description": "Lays a story ` +
				`out.", "messages": ` + messages + `, "tagline": "Scene by scene."}}}`,

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/TwoValues",

			body: whole + whole,

			expectStatus: http.StatusBadRequest,
		},
		{
			// A refused version names the field that breaks a rule, by its path in the body.
			name: "Error/InvalidFields",

			body: whole,

			serviceRequest: published(func(*core.PluginVersionPublishRequest) {}),
			serviceErr:     errors.Join(validator.New().Struct(&untitled), core.ErrInvalidRequest),

			expectStatus:   http.StatusUnprocessableEntity,
			expectResponse: invalidFieldsProblem(map[string]any{"languages[en].title": "required"}),
		},
		{
			// A document the database would not store as it is names its field and the rule it broke.
			name: "Error/InvalidDocument",

			body: whole,

			serviceRequest: published(func(*core.PluginVersionPublishRequest) {}),
			serviceErr:     &core.FieldError{Path: "languages[en].messages", Rule: "maxbytes"},

			expectStatus:   http.StatusUnprocessableEntity,
			expectResponse: invalidFieldsProblem(map[string]any{"languages[en].messages": "maxbytes"}),
		},
		{
			// A refused proof names the proof member, as a member that breaks a rule is named. Here no key
			// vouches for the proof the body carries.
			name: "Error/Proof",

			body: proven,

			serviceRequest: published(func(request *core.PluginVersionPublishRequest) { request.Proof = proof }),
			serviceErr:     core.ErrPluginProof,

			expectStatus:   http.StatusUnprocessableEntity,
			expectResponse: invalidFieldsProblem(map[string]any{"proof": "pluginproof"}),
		},
		{
			// Wiring case: a sentinel only pluginErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests"). It is also what this route answers for a version the registry
			// holds.
			name: "Error/VersionExists",

			body: whole,

			serviceRequest: published(func(*core.PluginVersionPublishRequest) {}),
			serviceErr:     core.ErrPluginVersionExists,

			expectStatus: http.StatusConflict,
		},
		{
			name: "Error/Internal",

			body: whole,

			serviceRequest: published(func(*core.PluginVersionPublishRequest) {}),
			serviceErr:     errFoo,

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestPluginVersionPublishService(t)

			if testCase.serviceRequest != nil {
				service.EXPECT().Exec(mock.Anything, testCase.serviceRequest).Return(testCase.serviceErr)
			}

			// The route verifies its publisher before the handler, which reads no user.
			request := httptest.NewRequestWithContext(
				t.Context(), http.MethodPost, "/v1/plugin/version", strings.NewReader(testCase.body),
			)

			if testCase.signed {
				token := key.Sign(t, servicejsonkeystest.TypRequestToken, key.RequestClaims([]byte(testCase.body)))
				request.Header.Set("Authorization", "Bearer "+token)
			}

			w := httptest.NewRecorder()
			gate(handlers.NewRestPluginVersionPublish(service, configtest.Logger)).ServeHTTP(w, request)

			response := w.Result()
			require.Equal(t, testCase.expectLocation, response.Header.Get("Location"))

			if testCase.expectStatus == http.StatusCreated {
				body, err := io.ReadAll(response.Body)
				require.NoError(t, errors.Join(err, response.Body.Close()))
				require.Empty(t, body)
			}

			requireResponse(t, response, testCase.expectStatus, testCase.expectResponse)
		})
	}
}
