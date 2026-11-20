package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestPluginVersionList(t *testing.T) {
	t.Parallel()

	target := "/v1/plugin/versions?plugin=scenes-composer&after=1.0.0&limit=10&offset=20"

	testCases := []struct {
		name string

		target string

		serviceMock *serviceMock[*core.PluginVersionPage]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			target: target,

			serviceMock: &serviceMock[*core.PluginVersionPage]{resp: &core.PluginVersionPage{
				Versions: []*core.PluginVersion{
					{Version: "1.2.0", HasMigration: true, SchemaChecksum: "first schema"},
					{Version: "1.10.0", SchemaChecksum: "second schema"},
				},
				Total: 5,
			}},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"versions": []any{
					map[string]any{"version": "1.2.0", "hasMigration": true, "schemaChecksum": "first schema"},
					map[string]any{"version": "1.10.0", "hasMigration": false, "schemaChecksum": "second schema"},
				},
				"total": 5.0,
			},
		},
		{
			name: "Success/Empty",

			target: target,

			serviceMock: &serviceMock[*core.PluginVersionPage]{
				resp: &core.PluginVersionPage{Versions: []*core.PluginVersion{}},
			},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"versions": []any{}, "total": 0.0},
		},
		{
			name: "Error/MalformedRequest",

			target: "/v1/plugin/versions?plugin=scenes-composer&after=1.0.0&limit=ten",

			expectStatus: http.StatusBadRequest,
		},
		{
			// Wiring case: a sentinel only pluginErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests"), though the route never produces it.
			name: "Error/VersionNotFound",

			target: target,

			serviceMock: &serviceMock[*core.PluginVersionPage]{err: core.ErrPluginVersionNotFound},

			expectStatus: http.StatusNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestPluginVersionListService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.PluginVersionListRequest{
						Plugin: "scenes-composer", After: "1.0.0", Limit: 10, Offset: 20,
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestPluginVersionList(service, configtest.Logger).
				ServeHTTP(w, newRequest(t, http.MethodGet, testCase.target, nil))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
