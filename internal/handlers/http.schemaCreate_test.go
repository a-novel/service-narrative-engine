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

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

func TestSchemaCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.SchemaCreateRequest
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
			name: "Success",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","projectID":"00000000-0000-0000-0000-000000000002","module":"namespace:module@v1.0.0","source":"USER","data":{"key":"value"}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaCreateRequest{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Module:    "namespace:module@v1.0.0",
					Source:    "USER",
					Data:      map[string]any{"key": "value"},
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
			expectStatus: http.StatusCreated,
		},
		{
			name: "Success/WithPreversion",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","projectID":"00000000-0000-0000-0000-000000000002","module":"namespace:module@v1.0.0-beta","source":"AI","data":{"foo":"bar"}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaCreateRequest{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Module:    "namespace:module@v1.0.0-beta",
					Source:    "AI",
					Data:      map[string]any{"foo": "bar"},
				},
				resp: &services.Schema{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Owner:            lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
					ModuleID:         "module",
					ModuleNamespace:  "namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "-beta",
					Source:           "AI",
					Data:             map[string]any{"foo": "bar"},
					CreatedAt:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expectResponse: map[string]any{
				"id":        "00000000-0000-0000-0000-000000000001",
				"projectID": "00000000-0000-0000-0000-000000000002",
				"owner":     "00000000-0000-0000-0000-000000000003",
				"module":    "namespace:module@v1.0.0-beta",
				"source":    "AI",
				"data":      map[string]any{"foo": "bar"},
				"createdAt": "2026-01-01T00:00:00Z",
			},
			expectStatus: http.StatusCreated,
		},
		{
			name: "Error/NoClaims",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","projectID":"00000000-0000-0000-0000-000000000002","module":"namespace:module@v1.0.0","source":"USER","data":{}}`),
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
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","projectID":"00000000-0000-0000-0000-000000000002","module":"namespace:module@v1.0.0","source":"USER","data":{}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaCreateRequest{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Module:    "namespace:module@v1.0.0",
					Source:    "USER",
					Data:      map[string]any{},
				},
				err: dao.ErrProjectSelectNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/ModuleNotFound",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","projectID":"00000000-0000-0000-0000-000000000002","module":"namespace:module@v1.0.0","source":"USER","data":{}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaCreateRequest{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Module:    "namespace:module@v1.0.0",
					Source:    "USER",
					Data:      map[string]any{},
				},
				err: dao.ErrModuleSelectNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/InvalidRequest",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","projectID":"00000000-0000-0000-0000-000000000002","module":"namespace:module@v1.0.0","source":"USER","data":{}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaCreateRequest{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Module:    "namespace:module@v1.0.0",
					Source:    "USER",
					Data:      map[string]any{},
				},
				err: services.ErrInvalidRequest,
			},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/UserDoesNotOwnProject",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","projectID":"00000000-0000-0000-0000-000000000002","module":"namespace:module@v1.0.0","source":"USER","data":{}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaCreateRequest{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Module:    "namespace:module@v1.0.0",
					Source:    "USER",
					Data:      map[string]any{},
				},
				err: services.ErrUserDoesNotOwnProject,
			},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/ModuleNotInProject",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","projectID":"00000000-0000-0000-0000-000000000002","module":"namespace:module@v1.0.0","source":"USER","data":{}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaCreateRequest{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Module:    "namespace:module@v1.0.0",
					Source:    "USER",
					Data:      map[string]any{},
				},
				err: services.ErrModuleNotInProject,
			},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/InternalError",

			request: httptest.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","projectID":"00000000-0000-0000-0000-000000000002","module":"namespace:module@v1.0.0","source":"USER","data":{}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			serviceMock: &serviceMock{
				req: &services.SchemaCreateRequest{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Module:    "namespace:module@v1.0.0",
					Source:    "USER",
					Data:      map[string]any{},
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockSchemaCreateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewSchemaCreate(service, config.LoggerDev)
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
