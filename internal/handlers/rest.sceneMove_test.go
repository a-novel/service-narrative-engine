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

func TestRestSceneMove(t *testing.T) {
	t.Parallel()

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPost, "/v1/scene/move", map[string]any{
			"projectID": projectID,
			"id":        sceneID,
			"beatID":    beatID,
			"position":  2,
		})
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[*core.Scene]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[*core.Scene]{resp: coreScene},

			expectStatus:   http.StatusOK,
			expectResponse: jsonScene,
		},
		{
			name: "Error/Anonymous",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return asUser(validRequest(t), nil)
			},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/MalformedRequest",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return newRequest(t, http.MethodPost, "/v1/scene/move", "not an object")
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			// Wiring case: a sentinel only composerErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests"), whether or not the route produces it.
			name: "Error/ProjectFull",

			request: validRequest,

			serviceMock: &serviceMock[*core.Scene]{err: core.ErrProjectFull},

			expectStatus: http.StatusConflict,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestSceneMoveService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.SceneMoveRequest{
						OwnerID: ownerID, ProjectID: projectID, ID: sceneID, BeatID: beatID, Position: 2,
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestSceneMove(service, configtest.Logger).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
