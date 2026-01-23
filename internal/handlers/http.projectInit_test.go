package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	authpkg "github.com/a-novel/service-authentication/v2/pkg"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

func TestProjectInit(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.ProjectInitRequest
		resp *services.Project
		err  error
	}

	testCases := []struct {
		name string

		request *http.Request
		claims  *authpkg.Claims

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"lang":"en","title":"Test Project","workflow":["step1","step2"]}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.ProjectInitRequest{
					Owner:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Lang:     "en",
					Title:    "Test Project",
					Workflow: []string{"step1", "step2"},
				},
				resp: &services.Project{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Owner:     uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Lang:      "en",
					Title:     "Test Project",
					Workflow:  []string{"step1", "step2"},
					CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expectResponse: map[string]any{
				"id":        "00000000-0000-0000-0000-000000000002",
				"owner":     "00000000-0000-0000-0000-000000000001",
				"lang":      "en",
				"title":     "Test Project",
				"workflow":  []any{"step1", "step2"},
				"createdAt": "2026-01-01T00:00:00Z",
				"updatedAt": "2026-01-01T00:00:00Z",
			},
			expectStatus: http.StatusCreated,
		},
		{
			name: "Error/NoClaims",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"lang":"en","title":"Test Project","workflow":["step1","step2"]}`),
			),

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/InvalidJSON",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{invalid`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/InvalidRequest",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"lang":"en","title":"Test Project","workflow":["step1","step2"]}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.ProjectInitRequest{
					Owner:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Lang:     "en",
					Title:    "Test Project",
					Workflow: []string{"step1", "step2"},
				},
				err: services.ErrInvalidRequest,
			},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/ModuleNotFound",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"lang":"en","title":"Test Project","workflow":["step1","step2"]}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.ProjectInitRequest{
					Owner:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Lang:     "en",
					Title:    "Test Project",
					Workflow: []string{"step1", "step2"},
				},
				err: dao.ErrModuleSelectNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/ProjectAlreadyExists",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"lang":"en","title":"Test Project","workflow":["step1","step2"]}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.ProjectInitRequest{
					Owner:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Lang:     "en",
					Title:    "Test Project",
					Workflow: []string{"step1", "step2"},
				},
				err: dao.ErrProjectInsertAlreadyExists,
			},

			expectStatus: http.StatusConflict,
		},
		{
			name: "Error/InternalError",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"lang":"en","title":"Test Project","workflow":["step1","step2"]}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.ProjectInitRequest{
					Owner:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Lang:     "en",
					Title:    "Test Project",
					Workflow: []string{"step1", "step2"},
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockProjectInitService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewProjectInit(service)
			w := httptest.NewRecorder()

			rCtx := testCase.request.Context()
			rCtx = authpkg.SetClaimsContext(rCtx, testCase.claims)

			handler.ServeHTTP(w, testCase.request.WithContext(rCtx))

			res := w.Result()

			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				data, err := io.ReadAll(res.Body)
				require.NoError(t, errors.Join(err, res.Body.Close()))

				var jsonRes any
				require.NoError(t, json.Unmarshal(data, &jsonRes))
				require.Equal(t, testCase.expectResponse, jsonRes)
			}

			service.AssertExpectations(t)
		})
	}
}
