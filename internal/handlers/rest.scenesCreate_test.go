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

func TestRestScenesCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPost, "/v1/scenes", map[string]any{
			"projectID": projectID,
			"scenes": []any{map[string]any{
				"beatID":      beatID,
				"position":    2,
				"title":       "Ouverture",
				"description": "Marie attend.",
				"prompt":      "Lent.",
			}},
		})
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[[]*core.Scene]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Scene]{resp: []*core.Scene{coreScene}},

			expectStatus:   http.StatusCreated,
			expectResponse: []any{jsonScene},
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
			name: "Error/ProjectNotFound",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Scene]{err: core.ErrProjectNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/BeatNotFound",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Scene]{err: core.ErrBeatNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/BeatFull",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Scene]{err: core.ErrProjectFull},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/InvalidRequest",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Scene]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Scene]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestScenesCreateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ScenesCreateRequest{
						OwnerID:   ownerID,
						ProjectID: projectID,
						Scenes: []core.ScenesCreateItem{
							{BeatID: beatID, Position: 2, Title: "Ouverture", Description: "Marie attend.", Prompt: "Lent."},
						},
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestScenesCreate(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			service.AssertExpectations(t)
		})
	}
}
