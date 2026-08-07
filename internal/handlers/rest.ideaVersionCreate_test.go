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

func TestRestIdeaVersionCreate(t *testing.T) {
	t.Parallel()

	body := `{"value":{"title":"The Nearer Light","genre":"speculative","seed":"The foghorn moves closer."}}`
	request := &core.IdeaVersionCreateRequest{
		Actor:     core.Actor{UserID: restOwnerID},
		ProjectID: restProjectID,
		Title:     "The Nearer Light",
		Genre:     "speculative",
		Seed:      "The foghorn moves closer.",
	}
	idea := &core.Idea{
		ProjectID: restProjectID,
		VersionID: restIdeaVersionID,
		Title:     request.Title,
		Genre:     request.Genre,
		Seed:      request.Seed,
		CreatedAt: restUpdatedAt,
	}
	errService := errors.New("service failure")

	testCases := []struct {
		name string

		path   string
		body   string
		claims *serviceauthentication.Claims
		setup  func(*handlersmocks.MockRestIdeaVersionCreateService)

		expectStatus int
		expectBody   string
	}{
		{
			name:   "Success",
			path:   "/v0/projects/" + restProjectID.String() + "/ideas",
			body:   body,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestIdeaVersionCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(idea, nil)
			},
			expectStatus: http.StatusCreated,
			expectBody: `{
				"id":"00000000-0000-0000-0000-000000000202",
				"projectID":"00000000-0000-0000-0000-000000000201",
				"value":{"title":"The Nearer Light","genre":"speculative","seed":"The foghorn moves closer."},
				"createdAt":"2026-08-07T10:01:00Z"
			}`,
		},
		{
			name:         "Error/InvalidProjectID",
			path:         "/v0/projects/nope/ideas",
			body:         body,
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "Error/InvalidJSON",
			path:         "/v0/projects/" + restProjectID.String() + "/ideas",
			body:         `{"value":`,
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:   "Error/NotFound",
			path:   "/v0/projects/" + restProjectID.String() + "/ideas",
			body:   body,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestIdeaVersionCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrProjectNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:   "Error/InvalidRequest",
			path:   "/v0/projects/" + restProjectID.String() + "/ideas",
			body:   body,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestIdeaVersionCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrInvalidRequest)
			},
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "Error/Service",
			path:   "/v0/projects/" + restProjectID.String() + "/ideas",
			body:   body,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestIdeaVersionCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, errService)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestIdeaVersionCreateService(t)
			if testCase.setup != nil {
				testCase.setup(service)
			}

			response := executeRestHandler(
				t,
				handlers.NewRestIdeaVersionCreate(service, config.LoggerDev),
				http.MethodPost,
				"/v0/projects/{projectID}/ideas",
				testCase.path,
				testCase.body,
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
