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

func TestRestInstanceDelete(t *testing.T) {
	t.Parallel()

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		target := "/v1/instance?projectID=" + projectID.String() + "&id=" + instanceID.String()

		return newRequest(t, http.MethodDelete, target, nil)
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
			name: "Error/MalformedID",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return newRequest(t, http.MethodDelete, "/v1/instance?projectID="+projectID.String()+"&id=nope", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/Default",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceDescriptor]{err: core.ErrInstanceDefault},

			expectStatus: http.StatusConflict,
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

			service := handlersmocks.NewMockRestInstanceDeleteService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.InstanceDeleteRequest{OwnerID: ownerID, ProjectID: projectID, ID: instanceID}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestInstanceDelete(service, configtest.Logger).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
