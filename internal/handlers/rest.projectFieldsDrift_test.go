package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	"github.com/a-novel/service-narrative-engine/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/v2/internal/handlers/mocks"
)

func TestRestProjectFieldsDrift(t *testing.T) {
	t.Parallel()

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
			// The fields carry the default instance, which no other read keeps in sync, and still no logo.
			name: "Success/DefaultInstance",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectFieldsDriftResult]{
				resp: &core.ProjectFieldsDriftResult{
					Projects: []*core.Project{coreProjectWithDefault}, Removed: []uuid.UUID{},
				},
			},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"projects": []any{jsonProjectFieldsWithDefault}, "removed": []any{},
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
			// Wiring case: a sentinel only projectErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests"), whether or not the route produces it.
			name: "Error/ProjectsFull",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectFieldsDriftResult]{err: core.ErrProjectsFull},

			expectStatus: http.StatusConflict,
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
			handlers.NewRestProjectFieldsDrift(service, configtest.Logger).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
