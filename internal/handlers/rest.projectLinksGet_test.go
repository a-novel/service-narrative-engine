package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	"github.com/a-novel/service-narrative-engine/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/v2/internal/handlers/mocks"
)

func TestRestProjectLinksGet(t *testing.T) {
	t.Parallel()

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodGet, "/v1/links?projectID="+projectID.String(), nil)
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[*core.ProjectLinks]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectLinks]{resp: &core.ProjectLinks{
				Links: json.RawMessage(`[{"relationship": "therefore"}]`), UpdatedAt: &preciseAt,
			}},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"links": []any{map[string]any{"relationship": "therefore"}}, "updatedAt": "2021-01-03T08:15:30.123456Z",
			},
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
			name: "Error/MalformedID",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return newRequest(t, http.MethodGet, "/v1/links?projectID=nope", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			// A project that never wrote its links answers an empty list and a null updatedAt.
			name: "Success/Unwritten",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectLinks]{resp: &core.ProjectLinks{Links: json.RawMessage(`[]`)}},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"links": []any{}, "updatedAt": nil},
		},
		{
			// Wiring case: a sentinel only instanceErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests").
			name: "Error/InstanceNotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectLinks]{err: core.ErrInstanceNotFound},

			expectStatus: http.StatusNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestProjectLinksGetService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ProjectLinksGetRequest{OwnerID: ownerID, ProjectID: projectID}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestProjectLinksGet(service, configtest.Logger).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
