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

func TestRestSceneCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPost, "/v1/scenes", map[string]any{
			"projectID":   projectID,
			"position":    2,
			"title":       "Ouverture",
			"color":       "#e0457b",
			"description": "Marie attend.",
			"prompt":      "Lent.",
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

			expectStatus:   http.StatusCreated,
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

				return newRequest(t, http.MethodPost, "/v1/scenes", "not an object")
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/NotFound",

			request: validRequest,

			serviceCalled: true,
			serviceErr:    core.ErrProjectNotFound,

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/ProjectFull",

			request: validRequest,

			serviceCalled: true,
			serviceErr:    core.ErrProjectFull,

			expectStatus: http.StatusUnprocessableEntity,
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

			service := handlersmocks.NewMockRestSceneCreateService(t)

			if testCase.serviceCalled {
				var resp *core.Scene
				if testCase.serviceErr == nil {
					resp = coreScene
				}

				service.EXPECT().
					Exec(mock.Anything, &core.SceneCreateRequest{
						OwnerID:     ownerID,
						ProjectID:   projectID,
						Position:    2,
						Title:       "Ouverture",
						Color:       "#e0457b",
						Description: "Marie attend.",
						Prompt:      "Lent.",
					}).
					Return(resp, testCase.serviceErr)
			}

			w := httptest.NewRecorder()
			handlers.NewRestSceneCreate(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
