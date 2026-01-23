package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	authpkg "github.com/a-novel/service-authentication/v2/pkg"

	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

func TestProjectList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.ProjectListRequest
		resp []*services.Project
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
				http.MethodGet,
				"/?limit=10&offset=0",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.ProjectListRequest{
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Limit:  10,
					Offset: 0,
				},
				resp: []*services.Project{
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						Owner:     uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						Lang:      "en",
						Title:     "Project One",
						Workflow:  []string{"step1", "step2"},
						CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
						Owner:     uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						Lang:      "fr",
						Title:     "Project Two",
						Workflow:  []string{"stepA"},
						CreatedAt: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expectResponse: []any{
				map[string]any{
					"id":        "00000000-0000-0000-0000-000000000002",
					"owner":     "00000000-0000-0000-0000-000000000001",
					"lang":      "en",
					"title":     "Project One",
					"workflow":  []any{"step1", "step2"},
					"createdAt": "2026-01-01T00:00:00Z",
					"updatedAt": "2026-01-02T00:00:00Z",
				},
				map[string]any{
					"id":        "00000000-0000-0000-0000-000000000003",
					"owner":     "00000000-0000-0000-0000-000000000001",
					"lang":      "fr",
					"title":     "Project Two",
					"workflow":  []any{"stepA"},
					"createdAt": "2026-01-03T00:00:00Z",
					"updatedAt": "2026-01-04T00:00:00Z",
				},
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Success/EmptyResult",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?limit=10&offset=0",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.ProjectListRequest{
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Limit:  10,
					Offset: 0,
				},
				resp: []*services.Project{},
			},

			expectResponse: []any{},
			expectStatus:   http.StatusOK,
		},
		{
			name: "Success/WithOffset",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?limit=5&offset=10",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.ProjectListRequest{
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Limit:  5,
					Offset: 10,
				},
				resp: []*services.Project{
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						Owner:     uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						Lang:      "en",
						Title:     "Project One",
						Workflow:  []string{"step1"},
						CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expectResponse: []any{
				map[string]any{
					"id":        "00000000-0000-0000-0000-000000000002",
					"owner":     "00000000-0000-0000-0000-000000000001",
					"lang":      "en",
					"title":     "Project One",
					"workflow":  []any{"step1"},
					"createdAt": "2026-01-01T00:00:00Z",
					"updatedAt": "2026-01-02T00:00:00Z",
				},
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Error/NoClaims",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?limit=10&offset=0",
				nil,
			),

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/InvalidRequest",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?limit=10&offset=0",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.ProjectListRequest{
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Limit:  10,
					Offset: 0,
				},
				err: services.ErrInvalidRequest,
			},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/InternalError",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?limit=10&offset=0",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			serviceMock: &serviceMock{
				req: &services.ProjectListRequest{
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Limit:  10,
					Offset: 0,
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockProjectListService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewProjectList(service)
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
