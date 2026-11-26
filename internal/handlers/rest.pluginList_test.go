package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	"github.com/a-novel/service-narrative-engine/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/v2/internal/handlers/mocks"
)

func TestRestPluginList(t *testing.T) {
	t.Parallel()

	target := "/v1/plugins?language=fr&limit=10&offset=20"

	testCases := []struct {
		name string

		target string

		// serviceRequest is the request the handler sends, the one of target when nil.
		serviceRequest *core.PluginListRequest
		serviceMock    *serviceMock[*core.PluginPage]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			target: target,

			serviceMock: &serviceMock[*core.PluginPage]{resp: &core.PluginPage{
				Plugins: []*core.Plugin{
					{Name: "scenes-composer", Version: "1.10.0", Title: "Compositeur de scènes", Description: "Compose."},
				},
				Total: 3,
			}},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"plugins": []any{map[string]any{
					"name": "scenes-composer", "version": "1.10.0", "title": "Compositeur de scènes",
					"description": "Compose.",
				}},
				"total": 3.0,
			},
		},
		{
			name: "Success/Empty",

			target: target,

			serviceMock: &serviceMock[*core.PluginPage]{resp: &core.PluginPage{Plugins: []*core.Plugin{}}},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"plugins": []any{}, "total": 0.0},
		},
		{
			name: "Success/NoParameter",

			target: "/v1/plugins",

			serviceRequest: &core.PluginListRequest{},
			serviceMock:    &serviceMock[*core.PluginPage]{resp: &core.PluginPage{Plugins: []*core.Plugin{}}},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"plugins": []any{}, "total": 0.0},
		},
		{
			name: "Error/MalformedRequest",

			target: "/v1/plugins?limit=ten",

			expectStatus: http.StatusBadRequest,
		},
		{
			// Wiring case: a sentinel only pluginErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests"), though the route never produces it.
			name: "Error/VersionNotFound",

			target: target,

			serviceMock: &serviceMock[*core.PluginPage]{err: core.ErrPluginVersionNotFound},

			expectStatus: http.StatusNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestPluginListService(t)

			if testCase.serviceMock != nil {
				serviceRequest := testCase.serviceRequest
				if serviceRequest == nil {
					serviceRequest = &core.PluginListRequest{Language: "fr", Limit: 10, Offset: 20}
				}

				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestPluginList(service, configtest.Logger).
				ServeHTTP(w, newRequest(t, http.MethodGet, testCase.target, nil))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
