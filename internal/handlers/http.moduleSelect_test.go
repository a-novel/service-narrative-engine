package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
	"github.com/a-novel/service-narrative-engine/internal/models"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

func TestModuleSelect(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.ModuleSelectRequest
		resp *services.Module
		err  error
	}

	testCases := []struct {
		name string

		request *http.Request

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?module=my-namespace:my-module@1.0.0",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleSelectRequest{
					Module: "my-namespace:my-module@1.0.0",
				},
				resp: &services.Module{
					ID:          "my-module",
					Namespace:   "my-namespace",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "A test module",
					Schema: jsonschema.Schema{
						Type: "object",
					},
					UI: models.ModuleUi{
						Component: "test-component",
						Target:    "test-target",
					},
					CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expectResponse: map[string]any{
				"id":          "my-module",
				"namespace":   "my-namespace",
				"version":     "1.0.0",
				"description": "A test module",
				"schema": map[string]any{
					"type": "object",
				},
				"ui": map[string]any{
					"component": "test-component",
					"params":    nil,
					"target":    "test-target",
				},
				"createdAt": "2026-01-01T00:00:00Z",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Success/WithPreversion",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?module=my-namespace:my-module@1.0.0-beta",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleSelectRequest{
					Module: "my-namespace:my-module@1.0.0-beta",
				},
				resp: &services.Module{
					ID:          "my-module",
					Namespace:   "my-namespace",
					Version:     "1.0.0",
					Preversion:  "beta",
					Description: "A test module",
					Schema: jsonschema.Schema{
						Type: "object",
					},
					UI: models.ModuleUi{
						Component: "test-component",
						Target:    "test-target",
					},
					CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expectResponse: map[string]any{
				"id":          "my-module",
				"namespace":   "my-namespace",
				"version":     "1.0.0",
				"preversion":  "beta",
				"description": "A test module",
				"schema": map[string]any{
					"type": "object",
				},
				"ui": map[string]any{
					"component": "test-component",
					"params":    nil,
					"target":    "test-target",
				},
				"createdAt": "2026-01-01T00:00:00Z",
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Error/NotFound",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?module=my-namespace:my-module@1.0.0",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleSelectRequest{
					Module: "my-namespace:my-module@1.0.0",
				},
				err: dao.ErrModuleSelectNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/InvalidRequest",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?module=my-namespace:my-module@1.0.0",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleSelectRequest{
					Module: "my-namespace:my-module@1.0.0",
				},
				err: services.ErrInvalidRequest,
			},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/InternalError",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?module=my-namespace:my-module@1.0.0",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleSelectRequest{
					Module: "my-namespace:my-module@1.0.0",
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockModuleSelectService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewModuleSelect(service, config.LoggerDev)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, testCase.request)

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
