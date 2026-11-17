package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestBeatCreate(t *testing.T) {
	t.Parallel()

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPost, "/v1/beat", map[string]any{
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

		serviceMock *serviceMock[[]*core.Beat]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Beat]{resp: []*core.Beat{coreBeat}},

			expectStatus:   http.StatusOK,
			expectResponse: jsonBeat,
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

				return newRequest(t, http.MethodPost, "/v1/beat", "not an object")
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			// Wiring case: a sentinel only composerErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests"), whether or not the route produces it.
			name: "Error/ProjectsFull",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Beat]{err: core.ErrProjectsFull},

			expectStatus:   http.StatusConflict,
			expectResponse: limitProblem("projects"),
		},
		{
			// The batch request names the title Beats[0].Title; the route names it as its own body does.
			name: "Error/InvalidFields",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Beat]{err: errors.Join(validator.New().Struct(&struct {
				Beats []struct {
					Title string `validate:"required"`
				} `validate:"dive"`
			}{Beats: make([]struct {
				Title string `validate:"required"`
			}, 1)}), core.ErrInvalidRequest)},

			expectStatus:   http.StatusUnprocessableEntity,
			expectResponse: invalidFieldsProblem(map[string]any{"title": "required"}),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestBeatCreateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.BeatsCreateRequest{
						OwnerID:   ownerID,
						ProjectID: projectID,
						Position:  2,
						Beats: []core.BeatsCreateItem{{
							Title:       "Ouverture",
							Color:       "#e0457b",
							Description: "Marie attend.",
							Prompt:      "Lent.",
						}},
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestBeatCreate(service, configtest.Logger).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
