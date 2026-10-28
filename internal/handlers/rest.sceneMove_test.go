package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestSceneMove(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPost, "/v1/scene/move", map[string]any{
			"projectID": projectID,
			"id":        sceneID,
			"after":     anchorID,
		})
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceCalled bool
		serviceErr    error

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceCalled: true,

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
			name: "Error/NotFound",

			request: validRequest,

			serviceCalled: true,
			serviceErr:    core.ErrSceneNotFound,

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/UnknownAnchor",

			request: validRequest,

			serviceCalled: true,
			serviceErr:    core.ErrInvalidRequest,

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceCalled: true,
			serviceErr:    errFoo,

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestSceneMoveService(t)

			if testCase.serviceCalled {
				var resp *core.Scene
				if testCase.serviceErr == nil {
					resp = coreScene
				}

				service.EXPECT().
					Exec(mock.Anything, &core.SceneMoveRequest{OwnerID: ownerID, ProjectID: projectID, ID: sceneID, After: anchorID}).
					Return(resp, testCase.serviceErr)
			}

			w := httptest.NewRecorder()
			handlers.NewRestSceneMove(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
