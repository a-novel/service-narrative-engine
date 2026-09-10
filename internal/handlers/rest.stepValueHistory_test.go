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

func TestRestStepValueHistory(t *testing.T) {
	t.Parallel()

	request := &core.StepValueHistoryRequest{
		Actor:     core.Actor{UserID: restOwnerID},
		ProjectID: restProjectID,
		Key:       "outline",
	}
	values := []*core.StepValue{{
		ID:        restStepValueID,
		ProjectID: restProjectID,
		Key:       request.Key,
		Value:     restStepValue,
		CreatedAt: restUpdatedAt,
	}}
	errService := errors.New("service failure")

	testCases := []struct {
		name string

		path   string
		claims *serviceauthentication.Claims
		setup  func(*handlersmocks.MockRestStepValueHistoryService)

		expectStatus int
		expectBody   string
	}{
		{
			name:   "Success",
			path:   "/v0/projects/" + restProjectID.String() + "/steps?key=outline",
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestStepValueHistoryService) {
				service.EXPECT().Exec(mock.Anything, request).Return(values, nil)
			},
			expectStatus: http.StatusOK,
			expectBody: `[{
				"id":"00000000-0000-0000-0000-000000000401",
				"projectID":"00000000-0000-0000-0000-000000000201",
				"key":"outline",
				"value":{"formerSchema":"intentionally opaque"},
				"createdAt":"2026-08-07T10:01:00Z"
			}]`,
		},
		{
			name:   "Success/Empty",
			path:   "/v0/projects/" + restProjectID.String() + "/steps?key=outline",
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestStepValueHistoryService) {
				service.EXPECT().Exec(mock.Anything, request).Return([]*core.StepValue{}, nil)
			},
			expectStatus: http.StatusOK,
			expectBody:   `[]`,
		},
		{
			name:         "Error/MissingClaims",
			path:         "/v0/projects/" + restProjectID.String() + "/steps?key=outline",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error/InvalidProjectID",
			path:         "/v0/projects/nope/steps?key=outline",
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:   "Error/InvalidRequest",
			path:   "/v0/projects/" + restProjectID.String() + "/steps?key=outline",
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestStepValueHistoryService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrInvalidRequest)
			},
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "Error/NotFound",
			path:   "/v0/projects/" + restProjectID.String() + "/steps?key=outline",
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestStepValueHistoryService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrProjectNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:   "Error/Service",
			path:   "/v0/projects/" + restProjectID.String() + "/steps?key=outline",
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestStepValueHistoryService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, errService)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestStepValueHistoryService(t)
			if testCase.setup != nil {
				testCase.setup(service)
			}

			response := executeRestHandler(
				t,
				handlers.NewRestStepValueHistory(service, config.LoggerDev),
				http.MethodGet,
				"/v0/projects/{projectID}/steps",
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
