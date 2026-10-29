package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestLayerList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodGet, "/v1/layers?projectID="+projectID.String()+"&limit=10&offset=20", nil)
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[*core.LayerPage]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[*core.LayerPage]{resp: &core.LayerPage{Layers: []*core.Layer{coreLayer}, Total: 3}},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"layers": []any{jsonLayer}, "total": 3.0},
		},
		{
			name: "Success/Empty",

			request: validRequest,

			serviceMock: &serviceMock[*core.LayerPage]{resp: &core.LayerPage{Layers: []*core.Layer{}}},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"layers": []any{}, "total": 0.0},
		},
		{
			name: "Error/Anonymous",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return asUser(validRequest(t), nil)
			},

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/MalformedRequest",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return newRequest(t, http.MethodGet, "/v1/layers?projectID=not-a-uuid", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.LayerPage]{err: core.ErrProjectNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/InvalidRequest",

			request: validRequest,

			serviceMock: &serviceMock[*core.LayerPage]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[*core.LayerPage]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestLayerListService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.LayerListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 10, Offset: 20}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestLayerList(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			service.AssertExpectations(t)
		})
	}
}
