package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
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

	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000042")
	projectID := uuid.MustParse("00000000-0000-0000-0000-000000000201")
	versionID := uuid.MustParse("00000000-0000-0000-0000-000000000202")
	createdAt := time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)

	service := handlersmocks.NewMockRestProjectCreateService(t)
	service.EXPECT().
		Exec(mock.Anything, &core.IdeaCreateRequest{
			Actor: core.Actor{UserID: ownerID},
			Title: "The Answering Light",
			Genre: "speculative",
			Seed:  "A foghorn answers from beneath the sea.",
		}).
		Return(&core.Idea{
			ProjectID:        projectID,
			VersionID:        versionID,
			OwnerID:          ownerID,
			Title:            "The Answering Light",
			Genre:            "speculative",
			Seed:             "A foghorn answers from beneath the sea.",
			ProjectCreatedAt: createdAt,
			CreatedAt:        createdAt,
		}, nil)

	handler := handlers.NewRestProjectCreate(service, config.LoggerDev)
	requestBody := `{
		"idea":{
			"title":"The Answering Light",
			"genre":"speculative",
			"seed":"A foghorn answers from beneath the sea."
		}
	}`
	requestContext := serviceauthentication.SetClaimsContext(
		t.Context(),
		&serviceauthentication.Claims{UserID: &ownerID},
	)
	request := httptest.NewRequestWithContext(
		requestContext,
		http.MethodPost,
		"/v0/projects",
		bytes.NewBufferString(requestBody),
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusCreated, response.Code)
	require.Equal(t, "/v0/projects/"+projectID.String(), response.Header().Get("Location"))
	require.JSONEq(t, `{
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
	}`, response.Body.String())
}
