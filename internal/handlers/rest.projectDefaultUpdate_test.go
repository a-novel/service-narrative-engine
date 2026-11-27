package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	"github.com/a-novel/service-narrative-engine/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/v2/internal/handlers/mocks"
)

func TestRestProjectDefaultUpdate(t *testing.T) {
	t.Parallel()

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPut, "/v1/project/default", map[string]any{
			"projectID": projectID.String(), "instanceID": instanceID.String(),
		})
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[*core.Project]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[*core.Project]{resp: coreProjectWithDefault},

			expectStatus:   http.StatusOK,
			expectResponse: jsonProjectWithDefault,
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

				return newRequest(t, http.MethodPut, "/v1/project/default", "not an object")
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			// Wiring case: a sentinel only instanceErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests").
			name: "Error/InstanceNotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.Project]{err: core.ErrInstanceNotFound},

			expectStatus: http.StatusNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestProjectDefaultUpdateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ProjectDefaultUpdateRequest{
						OwnerID: ownerID, ProjectID: projectID, InstanceID: instanceID,
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestProjectDefaultUpdate(service, configtest.Logger).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
