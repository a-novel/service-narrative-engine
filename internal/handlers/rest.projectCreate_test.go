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

func TestRestProjectCreate(t *testing.T) {
	t.Parallel()

	requestBody := `{
		"idea":{
			"title":"The Answering Light",
			"genre":"speculative",
			"seed":"A foghorn answers from beneath the sea."
		}
	}`
	request := &core.IdeaCreateRequest{
		Actor: core.Actor{UserID: restOwnerID},
		Title: "The Answering Light",
		Genre: "speculative",
		Seed:  "A foghorn answers from beneath the sea.",
	}
	idea := &core.Idea{
		ProjectID:        restProjectID,
		VersionID:        restIdeaVersionID,
		OwnerID:          restOwnerID,
		Title:            request.Title,
		Genre:            request.Genre,
		Seed:             request.Seed,
		ProjectCreatedAt: restCreatedAt,
		CreatedAt:        restCreatedAt,
	}
	errService := errors.New("service failure")

	testCases := []struct {
		name string

		body   string
		claims *serviceauthentication.Claims
		setup  func(*handlersmocks.MockRestProjectCreateService)

		expectStatus   int
		expectLocation string
		expectBody     string
	}{
		{
			name:   "Success",
			body:   requestBody,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestProjectCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(idea, nil)
			},
			expectStatus:   http.StatusCreated,
			expectLocation: "/v0/projects/" + restProjectID.String(),
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
				"stepValues":[],
				"manuscript":null
			}`,
		},
		{
			name:         "Error/MissingClaims",
			body:         requestBody,
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error/Anonymous",
			body:         requestBody,
			claims:       &serviceauthentication.Claims{},
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error/InvalidJSON",
			body:         `{"idea":`,
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "Error/MultipleDocuments",
			body:         requestBody + `{}`,
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "Error/UnknownField",
			body:         `{"idea":{"title":"Title","genre":"genre","seed":"seed"},"model":"client-owned"}`,
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:   "Error/InvalidRequest",
			body:   requestBody,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestProjectCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrInvalidRequest)
			},
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "Error/Service",
			body:   requestBody,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestProjectCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, errService)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestProjectCreateService(t)
			if testCase.setup != nil {
				testCase.setup(service)
			}

			response := executeRestHandler(
				t,
				handlers.NewRestProjectCreate(service, config.LoggerDev),
				http.MethodPost,
				"/v0/projects",
				"/v0/projects",
				testCase.body,
				testCase.claims,
				nil,
			)

			require.Equal(t, testCase.expectStatus, response.Code)
			require.Equal(t, testCase.expectLocation, response.Header().Get("Location"))

			if testCase.expectBody != "" {
				require.JSONEq(t, testCase.expectBody, response.Body.String())
			}

			service.AssertExpectations(t)
		})
	}
}
