package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

func TestModuleListVersions(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.ModuleListVersionsRequest
		resp []*services.ModuleVersion
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
				"/?id=my-module&namespace=my-namespace&limit=10&offset=0",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleListVersionsRequest{
					ID:        "my-module",
					Namespace: "my-namespace",
					Limit:     10,
					Offset:    0,
				},
				resp: []*services.ModuleVersion{
					{
						Version:    "1.0.0",
						Preversion: "",
						CreatedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					},
					{
						Version:    "1.1.0",
						Preversion: "beta",
						CreatedAt:  time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expectResponse: []any{
				map[string]any{
					"version":   "1.0.0",
					"createdAt": "2026-01-01T00:00:00Z",
				},
				map[string]any{
					"version":    "1.1.0",
					"preversion": "beta",
					"createdAt":  "2026-01-02T00:00:00Z",
				},
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Success/WithVersion",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?id=my-module&namespace=my-namespace&version=1.0",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleListVersionsRequest{
					ID:        "my-module",
					Namespace: "my-namespace",
					Version:   "1.0",
				},
				resp: []*services.ModuleVersion{
					{
						Version:    "1.0.0",
						Preversion: "",
						CreatedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					},
					{
						Version:    "1.0.1",
						Preversion: "",
						CreatedAt:  time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expectResponse: []any{
				map[string]any{
					"version":   "1.0.0",
					"createdAt": "2026-01-01T00:00:00Z",
				},
				map[string]any{
					"version":   "1.0.1",
					"createdAt": "2026-01-03T00:00:00Z",
				},
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Success/WithPreversion",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?id=my-module&namespace=my-namespace&preversion=true",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleListVersionsRequest{
					ID:         "my-module",
					Namespace:  "my-namespace",
					Preversion: true,
				},
				resp: []*services.ModuleVersion{
					{
						Version:    "1.0.0",
						Preversion: "alpha",
						CreatedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expectResponse: []any{
				map[string]any{
					"version":    "1.0.0",
					"preversion": "alpha",
					"createdAt":  "2026-01-01T00:00:00Z",
				},
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Success/EmptyResult",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?id=my-module&namespace=my-namespace",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleListVersionsRequest{
					ID:        "my-module",
					Namespace: "my-namespace",
				},
				resp: []*services.ModuleVersion{},
			},

			expectResponse: []any{},
			expectStatus:   http.StatusOK,
		},
		{
			name: "Error/InvalidRequest",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?id=my-module&namespace=my-namespace",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleListVersionsRequest{
					ID:        "my-module",
					Namespace: "my-namespace",
				},
				err: services.ErrInvalidRequest,
			},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/InternalError",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?id=my-module&namespace=my-namespace",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleListVersionsRequest{
					ID:        "my-module",
					Namespace: "my-namespace",
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockModuleListVersionsService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewModuleListVersions(service, config.LoggerDev)
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
