package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	"github.com/a-novel/service-narrative-engine/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/v2/internal/handlers/mocks"
)

func TestRestPluginManifestGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	target := "/v1/plugin/version?plugin=scenes-composer&version=1.10.0"

	manifest := &core.PluginManifest{
		Plugin: "scenes-composer", Version: "1.10.0", Schema: []byte(`{"type": "object"}`),
		Exports: []byte(`{"scenes": {"id": "/id"}}`), HasMigration: true, SchemaChecksum: "schema checksum",
		UIChecksum: "ui checksum", PublishedAt: createdAt,
	}

	signed := *manifest
	signed.Proof, signed.APIKeyID = new("header.claims.signature"), new("publisher-key")

	testCases := []struct {
		name string

		target string

		serviceMock *serviceMock[*core.PluginManifest]

		expectStatus   int
		expectResponse any
	}{
		{
			// A version published without a key carries neither a proof nor the ID of a key.
			name: "Success",

			target: target,

			serviceMock: &serviceMock[*core.PluginManifest]{resp: manifest},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"plugin": "scenes-composer", "version": "1.10.0", "schema": map[string]any{"type": "object"},
				"exports": map[string]any{"scenes": map[string]any{"id": "/id"}}, "hasMigration": true,
				"schemaChecksum": "schema checksum", "uiChecksum": "ui checksum",
				"publishedAt": "2021-01-01T00:00:00Z",
			},
		},
		{
			name: "Success/Proof",

			target: target,

			serviceMock: &serviceMock[*core.PluginManifest]{resp: &signed},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"plugin": "scenes-composer", "version": "1.10.0", "schema": map[string]any{"type": "object"},
				"exports": map[string]any{"scenes": map[string]any{"id": "/id"}}, "hasMigration": true,
				"schemaChecksum": "schema checksum", "uiChecksum": "ui checksum",
				"proof": "header.claims.signature", "apiKeyID": "publisher-key",
				"publishedAt": "2021-01-01T00:00:00Z",
			},
		},
		{
			name: "Error/MalformedRequest",

			// The decoder refuses a parameter the endpoint does not take.
			target: target + "&unknown=1",

			expectStatus: http.StatusBadRequest,
		},
		// This handler tests pluginErrors sentinel by sentinel (see CONTRIBUTING, "Handler tests").
		{
			name: "Error/InvalidRequest",

			target: target,

			serviceMock: &serviceMock[*core.PluginManifest]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/VersionNotFound",

			target: target,

			serviceMock: &serviceMock[*core.PluginManifest]{err: core.ErrPluginVersionNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/ModuleNotFound",

			target: target,

			serviceMock: &serviceMock[*core.PluginManifest]{err: core.ErrPluginModuleNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/MessagesNotFound",

			target: target,

			serviceMock: &serviceMock[*core.PluginManifest]{err: core.ErrPluginMessagesNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/VersionExists",

			target: target,

			serviceMock: &serviceMock[*core.PluginManifest]{err: core.ErrPluginVersionExists},

			expectStatus: http.StatusConflict,
		},
		{
			name: "Error/Internal",

			target: target,

			serviceMock: &serviceMock[*core.PluginManifest]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestPluginManifestGetService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.PluginManifestGetRequest{Plugin: "scenes-composer", Version: "1.10.0"}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestPluginManifestGet(service, configtest.Logger).
				ServeHTTP(w, newRequest(t, http.MethodGet, testCase.target, nil))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
