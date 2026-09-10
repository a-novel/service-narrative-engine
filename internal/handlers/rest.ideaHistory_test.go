package handlers_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestIdeaHistory(t *testing.T) {
	t.Parallel()

	request := &core.IdeaHistoryRequest{
		Actor:     core.Actor{UserID: restOwnerID},
		ProjectID: restProjectID,
	}
	ideas := []*core.Idea{{
		ProjectID: restProjectID,
		VersionID: restIdeaVersionID,
		Title:     "The Answering Light",
		Genre:     "speculative",
		Seed:      "A foghorn answers from beneath the sea.",
		CreatedAt: restCreatedAt,
	}}
	errService := errors.New("service failure")

	testCases := []struct {
		name string

		path   string
		claims *serviceauthentication.Claims
		setup  func(*handlersmocks.MockRestIdeaHistoryService)

		expectStatus int
		expectBody   string
	}{
		{
			name:   "Success",
			path:   "/v0/projects/" + restProjectID.String() + "/ideas",
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestIdeaHistoryService) {
				service.EXPECT().Exec(mock.Anything, request).Return(ideas, nil)
			},
			expectStatus: http.StatusOK,
			expectBody: `[{
				"id":"00000000-0000-0000-0000-000000000202",
				"projectID":"00000000-0000-0000-0000-000000000201",
				"value":{
					"title":"The Answering Light",
					"genre":"speculative",
					"seed":"A foghorn answers from beneath the sea."
				},
				"createdAt":"2026-08-07T10:00:00Z"
			}]`,
		},
		{
			name:   "Success/Empty",
			path:   "/v0/projects/" + restProjectID.String() + "/ideas",
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestIdeaHistoryService) {
				service.EXPECT().Exec(mock.Anything, request).Return([]*core.Idea{}, nil)
			},
			expectStatus: http.StatusOK,
			expectBody:   `[]`,
		},
		{
			name:         "Error/MissingClaims",
			path:         "/v0/projects/" + restProjectID.String() + "/ideas",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error/InvalidProjectID",
			path:         "/v0/projects/nope/ideas",
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:   "Error/NotFound",
			path:   "/v0/projects/" + restProjectID.String() + "/ideas",
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestIdeaHistoryService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrProjectNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:   "Error/Service",
			path:   "/v0/projects/" + restProjectID.String() + "/ideas",
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestIdeaHistoryService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, errService)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestIdeaHistoryService(t)
			if testCase.setup != nil {
				testCase.setup(service)
			}

			response := executeRestHandler(
				t,
				handlers.NewRestIdeaHistory(service, config.LoggerDev),
				http.MethodGet,
				"/v0/projects/{projectID}/ideas",
				testCase.path,
				"",
				testCase.claims,
				nil,
			)

			require.Equal(t, testCase.expectStatus, response.Code)

			if testCase.expectBody != "" {
				require.JSONEq(t, testCase.expectBody, response.Body.String())
			}

			service.AssertExpectations(t)
		})
	}
}
