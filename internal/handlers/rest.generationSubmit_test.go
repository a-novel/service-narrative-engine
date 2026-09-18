package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/logging"
	loggingpresets "github.com/a-novel-kit/golib/logging/presets"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestGenerationSubmit(t *testing.T) {
	t.Parallel()

	requestBody := `{
		"instructions":"Continue the scene without resolving it.",
		"input":{"paragraph":"The door opened."},
		"context":{"outline":{"beat":"refusal"}},
		"outputSchema":{"type":"object","required":["paragraph"]}
	}`
	request := &core.GenerationSubmitRequest{
		Actor:          core.Actor{UserID: restOwnerID},
		ProjectID:      restProjectID,
		IdempotencyKey: "draft-42",
		Instructions:   "Continue the scene without resolving it.",
		Input:          json.RawMessage(`{"paragraph":"The door opened."}`),
		Context:        json.RawMessage(`{"outline":{"beat":"refusal"}}`),
		OutputSchema:   json.RawMessage(`{"type":"object","required":["paragraph"]}`),
	}
	generation := &core.Generation{
		ID:          restGenerationID,
		Status:      core.GenerationStatusPending,
		MaxAttempts: 2,
		CreatedAt:   restCreatedAt,
		UpdatedAt:   restCreatedAt,
	}
	errService := errors.New("service failure")

	type testCase struct {
		name string

		path           string
		body           string
		claims         *serviceauthentication.Claims
		idempotencyKey string
		setup          func(*handlersmocks.MockRestGenerationSubmitService)
		logger         logging.Log
		logOutput      *bytes.Buffer
		logExcludes    string

		expectStatus     int
		expectLocation   string
		expectRetryAfter string
		expectBody       string
	}

	testCases := []testCase{
		{
			name:           "Success/Created",
			path:           "/v0/projects/" + restProjectID.String() + "/generations",
			body:           requestBody,
			claims:         &serviceauthentication.Claims{UserID: &restOwnerID},
			idempotencyKey: "draft-42",
			setup: func(service *handlersmocks.MockRestGenerationSubmitService) {
				service.EXPECT().Exec(mock.Anything, request).Return(&core.GenerationSubmitResult{
					Created:    true,
					Generation: generation,
				}, nil)
			},
			expectStatus:     http.StatusAccepted,
			expectLocation:   "/v0/projects/" + restProjectID.String() + "/generations/" + restGenerationID.String(),
			expectRetryAfter: "5",
			expectBody: `{
				"id":"00000000-0000-0000-0000-000000000601",
				"status":"pending",
				"attempt":0,
				"maxAttempts":2,
				"proposal":null,
				"failure":null,
				"createdAt":"2026-08-07T10:00:00Z",
				"updatedAt":"2026-08-07T10:00:00Z",
				"settledAt":null,
				"expiresAt":null
			}`,
		},
		{
			name:           "Success/Replay",
			path:           "/v0/projects/" + restProjectID.String() + "/generations",
			body:           requestBody,
			claims:         &serviceauthentication.Claims{UserID: &restOwnerID},
			idempotencyKey: "draft-42",
			setup: func(service *handlersmocks.MockRestGenerationSubmitService) {
				service.EXPECT().Exec(mock.Anything, request).Return(&core.GenerationSubmitResult{
					Created:    false,
					Generation: generation,
				}, nil)
			},
			expectStatus:   http.StatusOK,
			expectLocation: "/v0/projects/" + restProjectID.String() + "/generations/" + restGenerationID.String(),
		},
		{
			name:           "Error/MissingClaims",
			path:           "/v0/projects/" + restProjectID.String() + "/generations",
			body:           requestBody,
			idempotencyKey: "draft-42",
			expectStatus:   http.StatusUnauthorized,
		},
		{
			name:           "Error/InvalidProjectID",
			path:           "/v0/projects/nope/generations",
			body:           requestBody,
			claims:         &serviceauthentication.Claims{UserID: &restOwnerID},
			idempotencyKey: "draft-42",
			expectStatus:   http.StatusBadRequest,
		},
		{
			name:         "Error/MissingIdempotencyKey",
			path:         "/v0/projects/" + restProjectID.String() + "/generations",
			body:         requestBody,
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:           "Error/InvalidJSON",
			path:           "/v0/projects/" + restProjectID.String() + "/generations",
			body:           `{"instructions":`,
			claims:         &serviceauthentication.Claims{UserID: &restOwnerID},
			idempotencyKey: "draft-42",
			expectStatus:   http.StatusBadRequest,
		},
		{
			name:           "Error/MultipleDocuments",
			path:           "/v0/projects/" + restProjectID.String() + "/generations",
			body:           requestBody + `{}`,
			claims:         &serviceauthentication.Claims{UserID: &restOwnerID},
			idempotencyKey: "draft-42",
			expectStatus:   http.StatusBadRequest,
		},
		{
			name:           "Error/Conflict",
			path:           "/v0/projects/" + restProjectID.String() + "/generations",
			body:           requestBody,
			claims:         &serviceauthentication.Claims{UserID: &restOwnerID},
			idempotencyKey: "draft-42",
			setup: func(service *handlersmocks.MockRestGenerationSubmitService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrGenerationConflict)
			},
			expectStatus: http.StatusConflict,
		},
		{
			name:           "Error/InvalidRequest",
			path:           "/v0/projects/" + restProjectID.String() + "/generations",
			body:           requestBody,
			claims:         &serviceauthentication.Claims{UserID: &restOwnerID},
			idempotencyKey: "draft-42",
			setup: func(service *handlersmocks.MockRestGenerationSubmitService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrInvalidRequest)
			},
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "Error/Provider",
			path:           "/v0/projects/" + restProjectID.String() + "/generations",
			body:           requestBody,
			claims:         &serviceauthentication.Claims{UserID: &restOwnerID},
			idempotencyKey: "draft-42",
			setup: func(service *handlersmocks.MockRestGenerationSubmitService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrGenerationRefused)
			},
			expectStatus: http.StatusBadGateway,
		},
		{
			name:           "Error/Service",
			path:           "/v0/projects/" + restProjectID.String() + "/generations",
			body:           requestBody,
			claims:         &serviceauthentication.Claims{UserID: &restOwnerID},
			idempotencyKey: "draft-42",
			setup: func(service *handlersmocks.MockRestGenerationSubmitService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, errService)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, hiddenControl := range []string{
		"model",
		"provider",
		"reasoning",
		"effort",
		"maxOutputTokens",
		"maxAttempts",
		"purpose",
	} {
		logOutput := new(bytes.Buffer)
		testCases = append(testCases, testCase{
			name: "Error/HiddenProviderControl/" + hiddenControl,
			path: "/v0/projects/" + restProjectID.String() + "/generations",
			body: fmt.Sprintf(
				`{"instructions":"Continue the scene.","input":{},`+
					`"context":{},"outputSchema":{},%q:"client-selected"}`,
				hiddenControl,
			),
			claims:         &serviceauthentication.Claims{UserID: &restOwnerID},
			idempotencyKey: "draft-42",
			logger:         &loggingpresets.LogLocal{Out: logOutput},
			logOutput:      logOutput,
			logExcludes:    hiddenControl,
			expectStatus:   http.StatusBadRequest,
		})
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestGenerationSubmitService(t)
			if testCase.setup != nil {
				testCase.setup(service)
			}

			logger := testCase.logger
			if logger == nil {
				logger = config.LoggerDev
			}

			response := executeRestHandler(
				t,
				handlers.NewRestGenerationSubmit(service, logger),
				http.MethodPost,
				"/v0/projects/{projectID}/generations",
				testCase.path,
				testCase.body,
				testCase.claims,
				map[string]string{"Idempotency-Key": testCase.idempotencyKey},
			)

			require.Equal(t, testCase.expectStatus, response.Code)
			require.Equal(t, testCase.expectLocation, response.Header().Get("Location"))
			require.Equal(t, testCase.expectRetryAfter, response.Header().Get("Retry-After"))

			if testCase.expectBody != "" {
				require.JSONEq(t, testCase.expectBody, response.Body.String())
			}

			if testCase.logOutput != nil {
				require.NotContains(t, testCase.logOutput.String(), testCase.logExcludes)
			}

			service.AssertExpectations(t)
		})
	}
}
