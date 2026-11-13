package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestProjectFieldsDrift(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	removedID := uuid.MustParse("00000000-0000-0000-0000-0000000000ff")

	held := map[string]string{projectID.String(): "2021-01-02T00:00:00.123456Z"}

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPost, "/v1/projects/fields/drift", map[string]any{"projects": held})
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[*core.ProjectFieldsDriftResult]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectFieldsDriftResult]{
				resp: &core.ProjectFieldsDriftResult{Projects: []*core.Project{coreProject}, Removed: []uuid.UUID{removedID}},
			},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"projects": []any{jsonProjectFields}, "removed": []any{removedID.String()},
			},
		},
		{
			name: "Success/NoDrift",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectFieldsDriftResult]{
				resp: &core.ProjectFieldsDriftResult{Projects: []*core.Project{}, Removed: []uuid.UUID{}},
			},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"projects": []any{}, "removed": []any{}},
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

				return newRequest(t, http.MethodPost, "/v1/projects/fields/drift", map[string]any{"projects": []string{}})
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/InvalidRequest",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectFieldsDriftResult]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectFieldsDriftResult]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestProjectFieldsDriftService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ProjectFieldsDriftRequest{OwnerID: ownerID, Projects: held}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestProjectFieldsDrift(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			service.AssertExpectations(t)
		})
	}
}
