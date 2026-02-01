package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

func TestModuleListIDs(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.ModuleListIDsRequest
		resp []string
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
				"/?namespace=agora&limit=10&offset=0",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleListIDsRequest{
					Namespace: "agora",
					Limit:     10,
					Offset:    0,
				},
				resp: []string{"concept", "draft", "idea"},
			},

			expectResponse: []any{"concept", "draft", "idea"},
			expectStatus:   http.StatusOK,
		},
		{
			name: "Success/WithOffset",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?namespace=agora&limit=10&offset=5",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleListIDsRequest{
					Namespace: "agora",
					Limit:     10,
					Offset:    5,
				},
				resp: []string{"draft", "idea"},
			},

			expectResponse: []any{"draft", "idea"},
			expectStatus:   http.StatusOK,
		},
		{
			name: "Success/EmptyResult",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?namespace=agora&limit=10",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleListIDsRequest{
					Namespace: "agora",
					Limit:     10,
				},
				resp: []string{},
			},

			expectResponse: []any{},
			expectStatus:   http.StatusOK,
		},
		{
			name: "Error/InvalidRequest",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?namespace=agora&limit=10",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleListIDsRequest{
					Namespace: "agora",
					Limit:     10,
				},
				err: services.ErrInvalidRequest,
			},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/InternalError",

			request: httptest.NewRequest(
				http.MethodGet,
				"/?namespace=agora&limit=10",
				nil,
			),

			serviceMock: &serviceMock{
				req: &services.ModuleListIDsRequest{
					Namespace: "agora",
					Limit:     10,
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockModuleListIDsService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewModuleListIDs(service, config.LoggerDev)
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
