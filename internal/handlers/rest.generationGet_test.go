package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestGenerationGet(t *testing.T) {
	t.Parallel()

	request := &core.GenerationGetRequest{
		Actor:     core.Actor{UserID: restOwnerID},
		ProjectID: restProjectID,
		ID:        restGenerationID,
	}
	settledAt := restUpdatedAt.Add(time.Minute)
	expiresAt := settledAt.Add(30 * 24 * time.Hour)
	succeeded := &core.Generation{
		ID:          restGenerationID,
		Status:      core.GenerationStatusSucceeded,
		Attempt:     1,
		MaxAttempts: 2,
		Proposal:    json.RawMessage(`{"paragraph":"The door opened."}`),
		CreatedAt:   restCreatedAt,
		UpdatedAt:   restUpdatedAt,
		SettledAt:   &settledAt,
		ExpiresAt:   &expiresAt,
	}
	failed := &core.Generation{
		ID:          restGenerationID,
		Status:      core.GenerationStatusFailed,
		Attempt:     2,
		MaxAttempts: 2,
		Failure:     "generation failed",
		CreatedAt:   restCreatedAt,
		UpdatedAt:   restUpdatedAt,
		SettledAt:   &settledAt,
		ExpiresAt:   &expiresAt,
	}
	errService := errors.New("service failure")

	testCases := []struct {
		name string

		path   string
		claims *serviceauthentication.Claims
		setup  func(*handlersmocks.MockRestGenerationGetService)

		expectStatus int
		expectBody   string
	}{
		{
			name:   "Success/Succeeded",
			path:   "/v0/projects/" + restProjectID.String() + "/generations/" + restGenerationID.String(),
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestGenerationGetService) {
				service.EXPECT().Exec(mock.Anything, request).Return(succeeded, nil)
			},
			expectStatus: http.StatusOK,
			expectBody: `{
				"id":"00000000-0000-0000-0000-000000000601",
				"status":"succeeded",
				"attempt":1,
				"maxAttempts":2,
				"proposal":{"paragraph":"The door opened."},
				"failure":null,
				"createdAt":"2026-08-07T10:00:00Z",
				"updatedAt":"2026-08-07T10:01:00Z",
				"settledAt":"2026-08-07T10:02:00Z",
				"expiresAt":"2026-09-06T10:02:00Z"
			}`,
		},
		{
			name:   "Success/Failed",
			path:   "/v0/projects/" + restProjectID.String() + "/generations/" + restGenerationID.String(),
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestGenerationGetService) {
				service.EXPECT().Exec(mock.Anything, request).Return(failed, nil)
			},
			expectStatus: http.StatusOK,
			expectBody: `{
				"id":"00000000-0000-0000-0000-000000000601",
				"status":"failed",
				"attempt":2,
				"maxAttempts":2,
				"proposal":null,
				"failure":"generation failed",
				"createdAt":"2026-08-07T10:00:00Z",
				"updatedAt":"2026-08-07T10:01:00Z",
				"settledAt":"2026-08-07T10:02:00Z",
				"expiresAt":"2026-09-06T10:02:00Z"
			}`,
		},
		{
			name:         "Error/MissingClaims",
			path:         "/v0/projects/" + restProjectID.String() + "/generations/" + restGenerationID.String(),
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error/InvalidProjectID",
			path:         "/v0/projects/nope/generations/" + restGenerationID.String(),
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "Error/InvalidGenerationID",
			path:         "/v0/projects/" + restProjectID.String() + "/generations/nope",
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:   "Error/NotFound",
			path:   "/v0/projects/" + restProjectID.String() + "/generations/" + restGenerationID.String(),
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestGenerationGetService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrGenerationNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:   "Error/ProviderContract",
			path:   "/v0/projects/" + restProjectID.String() + "/generations/" + restGenerationID.String(),
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestGenerationGetService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrGenerationStatusUnknown)
			},
			expectStatus: http.StatusBadGateway,
		},
		{
			name:   "Error/Service",
			path:   "/v0/projects/" + restProjectID.String() + "/generations/" + restGenerationID.String(),
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestGenerationGetService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, errService)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestGenerationGetService(t)
			if testCase.setup != nil {
				testCase.setup(service)
			}

			response := executeRestHandler(
				t,
				handlers.NewRestGenerationGet(service, config.LoggerDev),
				http.MethodGet,
				"/v0/projects/{projectID}/generations/{generationID}",
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
