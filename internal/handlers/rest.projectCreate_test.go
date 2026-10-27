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

func TestRestProjectCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPost, "/v1/projects", map[string]any{
			"title":    "Les Heures claires",
			"premise":  "Marie attend.",
			"language": "fr",
			"scenes":   []any{map[string]any{"title": "Ouverture", "color": "#e0457b"}},
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
			expectResponse: withComposer(jsonProject, []any{jsonScene}, []any{jsonLayer}, []any{}),
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

				return newRequest(t, http.MethodPost, "/v1/projects", "not an object")
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/InvalidRequest",

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

			service := handlersmocks.NewMockRestProjectCreateService(t)

			if testCase.serviceCalled {
				var resp *core.ProjectComposer
				if testCase.serviceErr == nil {
					resp = &core.ProjectComposer{
						Project: coreProject, Scenes: []*core.Scene{coreScene}, Layers: []*core.Layer{coreLayer},
					}
				}

				service.EXPECT().
					Exec(mock.Anything, &core.ProjectCreateRequest{
						OwnerID:  ownerID,
						Title:    "Les Heures claires",
						Premise:  "Marie attend.",
						Language: "fr",
						Scenes:   []core.ProjectCreateScene{{Title: "Ouverture", Color: "#e0457b"}},
					}).
					Return(resp, testCase.serviceErr)
			}

			w := httptest.NewRecorder()
			handlers.NewRestProjectCreate(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
