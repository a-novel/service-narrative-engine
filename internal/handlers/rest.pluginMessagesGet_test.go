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

func TestRestPluginMessagesGet(t *testing.T) {
	t.Parallel()

	target := "/v1/plugin/version/messages?plugin=scenes-composer&version=1.10.0&language=fr"

	testCases := []struct {
		name string

		target string

		serviceMock *serviceMock[*core.PluginMessages]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			target: target,

			serviceMock: &serviceMock[*core.PluginMessages]{resp: &core.PluginMessages{
				Language: "fr", Title: "Compositeur de scènes", Description: "Compose.",
				Messages: []byte(`{"beats": "Temps"}`),
			}},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"language": "fr", "title": "Compositeur de scènes", "description": "Compose.",
				"messages": map[string]any{"beats": "Temps"},
			},
		},
		{
			name: "Error/MalformedRequest",

			// The decoder refuses a parameter the endpoint does not take.
			target: target + "&unknown=1",

			expectStatus: http.StatusBadRequest,
		},
		{
			// Wiring case: a sentinel only pluginErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests").
			name: "Error/MessagesNotFound",

			target: target,

			serviceMock: &serviceMock[*core.PluginMessages]{err: core.ErrPluginMessagesNotFound},

			expectStatus: http.StatusNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestPluginMessagesGetService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.PluginMessagesGetRequest{
						Plugin: "scenes-composer", Version: "1.10.0", Language: "fr",
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestPluginMessagesGet(service, configtest.Logger).
				ServeHTTP(w, newRequest(t, http.MethodGet, testCase.target, nil))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
