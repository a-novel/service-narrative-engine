package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestGenerationSubmit(t *testing.T) {
	t.Parallel()

	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000042")
	projectID := uuid.MustParse("00000000-0000-0000-0000-000000000201")
	generationID := uuid.MustParse("00000000-0000-0000-0000-000000000601")
	createdAt := time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)
	requestBody := `{
		"instructions":"Continue the scene without resolving it.",
		"input":{"paragraph":"The door opened."},
		"context":{"outline":{"beat":"refusal"}},
		"outputSchema":{"type":"object","required":["paragraph"]}
	}`

	t.Run("Created", func(t *testing.T) {
		t.Parallel()

		service := handlersmocks.NewMockRestGenerationSubmitService(t)
		service.EXPECT().
			Exec(mock.Anything, &core.GenerationSubmitRequest{
				Actor:          core.Actor{UserID: ownerID},
				ProjectID:      projectID,
				IdempotencyKey: "draft-42",
				Instructions:   "Continue the scene without resolving it.",
				Input:          json.RawMessage(`{"paragraph":"The door opened."}`),
				Context:        json.RawMessage(`{"outline":{"beat":"refusal"}}`),
				OutputSchema:   json.RawMessage(`{"type":"object","required":["paragraph"]}`),
			}).
			Return(&core.GenerationSubmitResult{
				Created: true,
				Generation: &core.Generation{
					ID:          generationID,
					Status:      core.GenerationStatusPending,
					MaxAttempts: 2,
					CreatedAt:   createdAt,
					UpdatedAt:   createdAt,
				},
			}, nil)

		response := executeGenerationSubmit(
			t,
			handlers.NewRestGenerationSubmit(service, config.LoggerDev),
			ownerID,
			projectID,
			"draft-42",
			requestBody,
		)

		require.Equal(t, http.StatusAccepted, response.Code)
		require.Equal(t, "5", response.Header().Get("Retry-After"))
		require.Equal(
			t,
			"/v0/projects/"+projectID.String()+"/generations/"+generationID.String(),
			response.Header().Get("Location"),
		)
		require.JSONEq(t, `{
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
		}`, response.Body.String())
	})

	t.Run("MissingIdempotencyKey", func(t *testing.T) {
		t.Parallel()

		response := executeGenerationSubmit(
			t,
			handlers.NewRestGenerationSubmit(
				handlersmocks.NewMockRestGenerationSubmitService(t),
				config.LoggerDev,
			),
			ownerID,
			projectID,
			"",
			requestBody,
		)

		require.Equal(t, http.StatusBadRequest, response.Code)
	})

	t.Run("Replay", func(t *testing.T) {
		t.Parallel()

		service := handlersmocks.NewMockRestGenerationSubmitService(t)
		service.EXPECT().
			Exec(mock.Anything, mock.AnythingOfType("*core.GenerationSubmitRequest")).
			Return(&core.GenerationSubmitResult{
				Created: false,
				Generation: &core.Generation{
					ID:          generationID,
					Status:      core.GenerationStatusPending,
					MaxAttempts: 2,
					CreatedAt:   createdAt,
					UpdatedAt:   createdAt,
				},
			}, nil)

		response := executeGenerationSubmit(
			t,
			handlers.NewRestGenerationSubmit(service, config.LoggerDev),
			ownerID,
			projectID,
			"draft-42",
			requestBody,
		)

		require.Equal(t, http.StatusOK, response.Code)
		require.Empty(t, response.Header().Get("Retry-After"))
		require.Equal(
			t,
			"/v0/projects/"+projectID.String()+"/generations/"+generationID.String(),
			response.Header().Get("Location"),
		)
	})

	t.Run("Conflict", func(t *testing.T) {
		t.Parallel()

		service := handlersmocks.NewMockRestGenerationSubmitService(t)
		service.EXPECT().
			Exec(mock.Anything, mock.AnythingOfType("*core.GenerationSubmitRequest")).
			Return(nil, core.ErrGenerationConflict)

		response := executeGenerationSubmit(
			t,
			handlers.NewRestGenerationSubmit(service, config.LoggerDev),
			ownerID,
			projectID,
			"draft-42",
			requestBody,
		)

		require.Equal(t, http.StatusConflict, response.Code)
	})
}

func executeGenerationSubmit(
	t *testing.T,
	handler http.Handler,
	ownerID uuid.UUID,
	projectID uuid.UUID,
	idempotencyKey string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	router.Post("/v0/projects/{projectID}/generations", handler.ServeHTTP)

	requestContext := serviceauthentication.SetClaimsContext(
		t.Context(),
		&serviceauthentication.Claims{UserID: &ownerID},
	)

	request := httptest.NewRequestWithContext(
		requestContext,
		http.MethodPost,
		"/v0/projects/"+projectID.String()+"/generations",
		bytes.NewBufferString(body),
	)
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}

	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	return response
}
