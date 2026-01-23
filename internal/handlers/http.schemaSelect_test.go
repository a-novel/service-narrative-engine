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

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

func TestSchemaSelect(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.SchemaSelectRequest
		resp *services.Schema
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
			name: "Success/ByID",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?id=00000000-0000-0000-0000-000000000001&projectID=00000000-0000-0000-0000-000000000002",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaSelectRequest{
					ID:        lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				},
				resp: &services.Schema{
					ID:              uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Owner:           lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
					ModuleID:        "module",
					ModuleNamespace: "namespace",
					ModuleVersion:   "1.0.0",
					Source:          "USER",
					Data:            map[string]any{"key": "value"},
					CreatedAt:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expectResponse: map[string]any{
				"id":        "00000000-0000-0000-0000-000000000001",
				"projectID": "00000000-0000-0000-0000-000000000002",
				"owner":     "00000000-0000-0000-0000-000000000003",
				"module":    "namespace:module@v1.0.0",
				"source":    "USER",
				"data":      map[string]any{"key": "value"},
				"createdAt": "2026-01-01T00:00:00Z",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Success/ByProjectAndModule",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?projectID=00000000-0000-0000-0000-000000000002&module=namespace:module@v1.0.0",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaSelectRequest{
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Module:    "namespace:module@v1.0.0",
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				},
				resp: &services.Schema{
					ID:              uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Owner:           lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
					ModuleID:        "module",
					ModuleNamespace: "namespace",
					ModuleVersion:   "1.0.0",
					Source:          "AI",
					Data:            map[string]any{"foo": "bar"},
					CreatedAt:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expectResponse: map[string]any{
				"id":        "00000000-0000-0000-0000-000000000001",
				"projectID": "00000000-0000-0000-0000-000000000002",
				"owner":     "00000000-0000-0000-0000-000000000003",
				"module":    "namespace:module@v1.0.0",
				"source":    "AI",
				"data":      map[string]any{"foo": "bar"},
				"createdAt": "2026-01-01T00:00:00Z",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Success/WithPreversion",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?projectID=00000000-0000-0000-0000-000000000002&module=namespace:module@v1.0.0-beta",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaSelectRequest{
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Module:    "namespace:module@v1.0.0-beta",
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				},
				resp: &services.Schema{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Owner:            lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
					ModuleID:         "module",
					ModuleNamespace:  "namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "-beta",
					Source:           "USER",
					Data:             map[string]any{"key": "value"},
					CreatedAt:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expectResponse: map[string]any{
				"id":        "00000000-0000-0000-0000-000000000001",
				"projectID": "00000000-0000-0000-0000-000000000002",
				"owner":     "00000000-0000-0000-0000-000000000003",
				"module":    "namespace:module@v1.0.0-beta",
				"source":    "USER",
				"data":      map[string]any{"key": "value"},
				"createdAt": "2026-01-01T00:00:00Z",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Error/NoClaims",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?id=00000000-0000-0000-0000-000000000001&projectID=00000000-0000-0000-0000-000000000002",
				nil,
			),

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/SchemaNotFound",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?id=00000000-0000-0000-0000-000000000001&projectID=00000000-0000-0000-0000-000000000002",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaSelectRequest{
					ID:        lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				},
				err: dao.ErrSchemaSelectNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/ProjectNotFound",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?id=00000000-0000-0000-0000-000000000001&projectID=00000000-0000-0000-0000-000000000002",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaSelectRequest{
					ID:        lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				},
				err: dao.ErrProjectSelectNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/InvalidRequest",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?id=00000000-0000-0000-0000-000000000001&projectID=00000000-0000-0000-0000-000000000002",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaSelectRequest{
					ID:        lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				},
				err: services.ErrInvalidRequest,
			},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/UserDoesNotOwnProject",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?id=00000000-0000-0000-0000-000000000001&projectID=00000000-0000-0000-0000-000000000002",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaSelectRequest{
					ID:        lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				},
				err: services.ErrUserDoesNotOwnProject,
			},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/InternalError",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?id=00000000-0000-0000-0000-000000000001&projectID=00000000-0000-0000-0000-000000000002",
				nil,
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaSelectRequest{
					ID:        lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockSchemaSelectService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewSchemaSelect(service)
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
