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

func TestRestProjectGet(t *testing.T) {
	t.Parallel()

	request := &core.ProjectGetRequest{
		Actor:     core.Actor{UserID: restOwnerID},
		ProjectID: restProjectID,
	}
	project := &core.ProjectSnapshot{
		ID:        restProjectID,
		CreatedAt: restCreatedAt,
		Idea: &core.Idea{
			ProjectID: restProjectID,
			VersionID: restIdeaVersionID,
			Title:     "The Answering Light",
			Genre:     "speculative",
			Seed:      "A foghorn answers from beneath the sea.",
			CreatedAt: restCreatedAt,
		},
		StepValues: []*core.StepValue{{
			ID:        restStepValueID,
			ProjectID: restProjectID,
			Key:       "outline",
			Value:     restStepValue,
			CreatedAt: restUpdatedAt,
		}},
		Manuscript: &core.Manuscript{
			ID:         restManuscriptID,
			ProjectID:  restProjectID,
			Manuscript: restManuscriptValue,
			CreatedAt:  restUpdatedAt,
		},
	}
	errService := errors.New("service failure")

	testCases := []struct {
		name string

		path   string
		claims *serviceauthentication.Claims
		setup  func(*handlersmocks.MockRestProjectGetService)

		expectStatus int
		expectBody   string
	}{
		{
			name:   "Success",
			path:   "/v0/projects/" + restProjectID.String(),
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestProjectGetService) {
				service.EXPECT().Exec(mock.Anything, request).Return(project, nil)
			},
			expectStatus: http.StatusOK,
			expectBody: `{
				"id":"00000000-0000-0000-0000-000000000201",
				"createdAt":"2026-08-07T10:00:00Z",
				"idea":{
					"id":"00000000-0000-0000-0000-000000000202",
					"projectID":"00000000-0000-0000-0000-000000000201",
					"value":{
						"title":"The Answering Light",
						"genre":"speculative",
						"seed":"A foghorn answers from beneath the sea."
					},
					"createdAt":"2026-08-07T10:00:00Z"
				},
				"stepValues":[{
					"id":"00000000-0000-0000-0000-000000000401",
					"projectID":"00000000-0000-0000-0000-000000000201",
					"key":"outline",
					"value":{"formerSchema":"intentionally opaque"},
					"createdAt":"2026-08-07T10:01:00Z"
				}],
				"manuscript":{
					"id":"00000000-0000-0000-0000-000000000501",
					"projectID":"00000000-0000-0000-0000-000000000201",
					"value":{"blocks":[{"type":"text","data":{"text":"The foghorn answered."},"metadata":{}}]},
					"createdAt":"2026-08-07T10:01:00Z"
				}
			}`,
		},
		{
			name:         "Error/MissingClaims",
			path:         "/v0/projects/" + restProjectID.String(),
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error/InvalidProjectID",
			path:         "/v0/projects/not-a-uuid",
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:   "Error/NotFound",
			path:   "/v0/projects/" + restProjectID.String(),
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestProjectGetService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrProjectNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:   "Error/Service",
			path:   "/v0/projects/" + restProjectID.String(),
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestProjectGetService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, errService)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestProjectGetService(t)
			if testCase.setup != nil {
				testCase.setup(service)
			}

			response := executeRestHandler(
				t,
				handlers.NewRestProjectGet(service, config.LoggerDev),
				http.MethodGet,
				"/v0/projects/{projectID}",
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
