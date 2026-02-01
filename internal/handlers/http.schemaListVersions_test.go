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

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

func TestSchemaListVersions(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.SchemaListVersionsRequest
		resp []*services.SchemaVersion
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
				"/?projectID=00000000-0000-0000-0000-000000000001&moduleID=my-module&moduleNamespace=my-namespace&limit=10&offset=0",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaListVersionsRequest{
					ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ModuleID:        "my-module",
					ModuleNamespace: "my-namespace",
					Limit:           10,
					Offset:          0,
				},
				resp: []*services.SchemaVersion{
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000010"),
						CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000011"),
						CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expectResponse: []any{
				map[string]any{
					"id":        "00000000-0000-0000-0000-000000000010",
					"createdAt": "2026-01-01T00:00:00Z",
				},
				map[string]any{
					"id":        "00000000-0000-0000-0000-000000000011",
					"createdAt": "2026-01-02T00:00:00Z",
				},
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Success/WithOffset",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?projectID=00000000-0000-0000-0000-000000000001&moduleID=my-module&moduleNamespace=my-namespace&limit=5&offset=10",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaListVersionsRequest{
					ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ModuleID:        "my-module",
					ModuleNamespace: "my-namespace",
					Limit:           5,
					Offset:          10,
				},
				resp: []*services.SchemaVersion{
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000020"),
						CreatedAt: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expectResponse: []any{
				map[string]any{
					"id":        "00000000-0000-0000-0000-000000000020",
					"createdAt": "2026-01-05T00:00:00Z",
				},
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Success/EmptyResult",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?projectID=00000000-0000-0000-0000-000000000001&moduleID=my-module&moduleNamespace=my-namespace&limit=10",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaListVersionsRequest{
					ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ModuleID:        "my-module",
					ModuleNamespace: "my-namespace",
					Limit:           10,
				},
				resp: []*services.SchemaVersion{},
			},

			expectResponse: []any{},
			expectStatus:   http.StatusOK,
		},
		{
			name: "Error/NoClaims",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?projectID=00000000-0000-0000-0000-000000000001&moduleID=my-module&moduleNamespace=my-namespace&limit=10",
				nil,
			),

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/ProjectNotFound",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?projectID=00000000-0000-0000-0000-000000000001&moduleID=my-module&moduleNamespace=my-namespace&limit=10",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaListVersionsRequest{
					ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ModuleID:        "my-module",
					ModuleNamespace: "my-namespace",
					Limit:           10,
				},
				err: dao.ErrProjectSelectNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/InvalidRequest",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?projectID=00000000-0000-0000-0000-000000000001&moduleID=my-module&moduleNamespace=my-namespace&limit=10",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaListVersionsRequest{
					ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ModuleID:        "my-module",
					ModuleNamespace: "my-namespace",
					Limit:           10,
				},
				err: services.ErrInvalidRequest,
			},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/UserDoesNotOwnProject",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?projectID=00000000-0000-0000-0000-000000000001&moduleID=my-module&moduleNamespace=my-namespace&limit=10",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaListVersionsRequest{
					ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ModuleID:        "my-module",
					ModuleNamespace: "my-namespace",
					Limit:           10,
				},
				err: services.ErrUserDoesNotOwnProject,
			},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/InternalError",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?projectID=00000000-0000-0000-0000-000000000001&moduleID=my-module&moduleNamespace=my-namespace&limit=10",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaListVersionsRequest{
					ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					UserID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ModuleID:        "my-module",
					ModuleNamespace: "my-namespace",
					Limit:           10,
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockSchemaListVersionsService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewSchemaListVersions(service, config.LoggerDev)
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
