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

func TestRestManuscriptCreate(t *testing.T) {
	t.Parallel()

	body := `{"value":{"blocks":[{"type":"text","data":{"text":"The foghorn answered."},"metadata":{}}]}}`
	request := &core.ManuscriptCreateRequest{
		Actor:      core.Actor{UserID: restOwnerID},
		ProjectID:  restProjectID,
		Manuscript: restManuscriptValue,
	}
	manuscript := &core.Manuscript{
		ID:         restManuscriptID,
		ProjectID:  restProjectID,
		Manuscript: restManuscriptValue,
		CreatedAt:  restUpdatedAt,
	}
	errService := errors.New("service failure")

	testCases := []struct {
		name string

		path   string
		body   string
		claims *serviceauthentication.Claims
		setup  func(*handlersmocks.MockRestManuscriptCreateService)

		expectStatus int
		expectBody   string
	}{
		{
			name:   "Success",
			path:   "/v0/projects/" + restProjectID.String() + "/manuscripts",
			body:   body,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestManuscriptCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(manuscript, nil)
			},
			expectStatus: http.StatusCreated,
			expectBody: `{
				"id":"00000000-0000-0000-0000-000000000501",
				"projectID":"00000000-0000-0000-0000-000000000201",
				"value":{"blocks":[{"type":"text","data":{"text":"The foghorn answered."},"metadata":{}}]},
				"createdAt":"2026-08-07T10:01:00Z"
			}`,
		},
		{
			name:         "Error/MissingClaims",
			path:         "/v0/projects/" + restProjectID.String() + "/manuscripts",
			body:         body,
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "Error/InvalidProjectID",
			path:         "/v0/projects/nope/manuscripts",
			body:         body,
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "Error/InvalidJSON",
			path:         "/v0/projects/" + restProjectID.String() + "/manuscripts",
			body:         `{"value":`,
			claims:       &serviceauthentication.Claims{UserID: &restOwnerID},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:   "Error/NotFound",
			path:   "/v0/projects/" + restProjectID.String() + "/manuscripts",
			body:   body,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestManuscriptCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrProjectNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:   "Error/InvalidRequest",
			path:   "/v0/projects/" + restProjectID.String() + "/manuscripts",
			body:   body,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestManuscriptCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, core.ErrInvalidRequest)
			},
			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "Error/Service",
			path:   "/v0/projects/" + restProjectID.String() + "/manuscripts",
			body:   body,
			claims: &serviceauthentication.Claims{UserID: &restOwnerID},
			setup: func(service *handlersmocks.MockRestManuscriptCreateService) {
				service.EXPECT().Exec(mock.Anything, request).Return(nil, errService)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestManuscriptCreateService(t)
			if testCase.setup != nil {
				testCase.setup(service)
			}

			response := executeRestHandler(
				t,
				handlers.NewRestManuscriptCreate(service, config.LoggerDev),
				http.MethodPost,
				"/v0/projects/{projectID}/manuscripts",
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
