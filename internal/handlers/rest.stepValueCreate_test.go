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

func TestRestStepValueCreate(t *testing.T) {
	t.Parallel()

	body := `{"key":"outline","value":{"formerSchema":"intentionally opaque"}}`
	request := &core.StepValueCreateRequest{
		Actor:     core.Actor{UserID: restOwnerID},
		ProjectID: restProjectID,
		Key:       "outline",
		Value:     restStepValue,
	}
	value := &core.StepValue{
		ID:        restStepValueID,
		ProjectID: restProjectID,
		Key:       request.Key,
		Value:     restStepValue,
		CreatedAt: restUpdatedAt,
	}
	errService := errors.New("service failure")

	testCases := []struct {
		name string

		path   string
		body   string
		claims *serviceauthentication.Claims
		setup  func(*handlersmocks.MockRestStepValueCreateService)

		expectStatus int
		expectBody   string
	}{
		{
			name:   "Success",
			path:   "/v0/projects/" + restProjectID.String() + "/steps",
			body:   body,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestStepValueCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(value, nil)
			},
			expectStatus: http.StatusCreated,
			expectBody: `{
				"id":"00000000-0000-0000-0000-000000000401",
				"projectID":"00000000-0000-0000-0000-000000000201",
				"key":"outline",
				"value":{"formerSchema":"intentionally opaque"},
				"createdAt":"2026-08-07T10:01:00Z"
			}`,
		},
		{
			name:         "Error/MissingClaims",
			path:         "/v0/projects/" + restProjectID.String() + "/steps",
			body:         body,
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error/InvalidProjectID",
			path:         "/v0/projects/nope/steps",
			body:         body,
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "Error/InvalidJSON",
			path:         "/v0/projects/" + restProjectID.String() + "/steps",
			body:         `{"key":"outline","value":`,
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:   "Error/NotFound",
			path:   "/v0/projects/" + restProjectID.String() + "/steps",
			body:   body,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestStepValueCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrProjectNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:   "Error/InvalidRequest",
			path:   "/v0/projects/" + restProjectID.String() + "/steps",
			body:   body,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestStepValueCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrInvalidRequest)
			},
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "Error/Service",
			path:   "/v0/projects/" + restProjectID.String() + "/steps",
			body:   body,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestStepValueCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, errService)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestStepValueCreateService(t)
			if testCase.setup != nil {
				testCase.setup(service)
			}

			response := executeRestHandler(
				t,
				handlers.NewRestStepValueCreate(service, config.LoggerDev),
				http.MethodPost,
				"/v0/projects/{projectID}/steps",
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
