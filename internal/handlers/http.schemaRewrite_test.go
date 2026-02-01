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

func TestSchemaRewrite(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
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
				http.MethodPut,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","data":{"key":"value"}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			serviceMock: &serviceMock{
				resp: &services.Schema{
					ID:              uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Owner:           lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
					ModuleID:        "module",
					ModuleNamespace: "namespace",
					ModuleVersion:   "1.0.0",
					Source:          "USER",
					Data:            map[string]any{"key": "value"},
					CreatedAt:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expectResponse: map[string]any{
				"createdAt": "2026-01-01T00:00:00Z",
				"data":      map[string]any{"key": "value"},
				"id":        "00000000-0000-0000-0000-000000000001",
				"module":    "namespace:module@v1.0.0",
				"owner":     "00000000-0000-0000-0000-000000000002",
				"projectID": "00000000-0000-0000-0000-000000000003",
				"source":    "USER",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Success/WithPreversion",

			request: httptest.NewRequest(
				http.MethodPut,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","data":{"foo":"bar"}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			serviceMock: &serviceMock{
				resp: &services.Schema{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Owner:            lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
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
				"createdAt": "2026-01-01T00:00:00Z",
				"data":      map[string]any{"foo": "bar"},
				"id":        "00000000-0000-0000-0000-000000000001",
				"module":    "namespace:module@v1.0.0-beta",
				"owner":     "00000000-0000-0000-0000-000000000002",
				"projectID": "00000000-0000-0000-0000-000000000003",
				"source":    "AI",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Error/NoClaims",

			request: httptest.NewRequest(
				http.MethodPut,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","data":{"key":"value"}}`),
			),

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/InvalidJSON",

			request: httptest.NewRequest(
				http.MethodPut,
				"/",
				strings.NewReader(`{invalid`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/SchemaNotFound",

			request: httptest.NewRequest(
				http.MethodPut,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","data":{"key":"value"}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			serviceMock: &serviceMock{
				err: dao.ErrSchemaSelectNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/ProjectNotFound",

			request: httptest.NewRequest(
				http.MethodPut,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","data":{"key":"value"}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			serviceMock: &serviceMock{
				err: dao.ErrProjectSelectNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/InvalidRequest",

			request: httptest.NewRequest(
				http.MethodPut,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","data":{"key":"value"}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			serviceMock: &serviceMock{
				err: services.ErrInvalidRequest,
			},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/UserDoesNotOwnProject",

			request: httptest.NewRequest(
				http.MethodPut,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","data":{"key":"value"}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			serviceMock: &serviceMock{
				err: services.ErrUserDoesNotOwnProject,
			},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/InternalError",

			request: httptest.NewRequest(
				http.MethodPut,
				"/",
				strings.NewReader(`{"id":"00000000-0000-0000-0000-000000000001","data":{"key":"value"}}`),
			),
			claims: &authpkg.Claims{
				UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
			},

			serviceMock: &serviceMock{
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockSchemaRewriteService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, mock.Anything).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewSchemaRewrite(service, config.LoggerDev)
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
