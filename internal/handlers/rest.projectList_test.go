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

func TestRestProjectList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodGet, "/v1/projects?limit=10&offset=20", nil)
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[*core.ProjectPage]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectPage]{
				resp: &core.ProjectPage{Projects: []*core.Project{coreProject}, Total: 3},
			},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"projects": []any{jsonProject}, "total": 3.0},
		},
		{
			name: "Success/Empty",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectPage]{resp: &core.ProjectPage{Projects: []*core.Project{}}},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"projects": []any{}, "total": 0.0},
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

				return newRequest(t, http.MethodGet, "/v1/projects?id=not-a-uuid", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/InvalidRequest",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectPage]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectPage]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestProjectListService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ProjectListRequest{OwnerID: ownerID, Limit: 10, Offset: 20}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestProjectList(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			service.AssertExpectations(t)
		})
	}
}
