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

func TestRestInstanceUpdate(t *testing.T) {
	t.Parallel()

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPut, "/v1/instance", map[string]any{
			"projectID": projectID.String(), "id": instanceID.String(), "name": "Outline",
		})
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[*core.InstanceDescriptor]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceDescriptor]{resp: coreDescriptor},

			expectStatus:   http.StatusOK,
			expectResponse: jsonDescriptor,
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

				return newRequest(t, http.MethodPut, "/v1/instance", "not an object")
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/ProjectBudget",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceDescriptor]{err: core.ErrProjectBudget},

			expectStatus:   http.StatusConflict,
			expectResponse: limitProblem("projectBytes"),
		},
		{
			// Wiring case: a sentinel only instanceErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests").
			name: "Error/InstanceNotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceDescriptor]{err: core.ErrInstanceNotFound},

			expectStatus: http.StatusNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestInstanceUpdateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.InstanceUpdateRequest{
						OwnerID: ownerID, ProjectID: projectID, ID: instanceID, Name: "Outline",
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestInstanceUpdate(service, configtest.Logger).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
