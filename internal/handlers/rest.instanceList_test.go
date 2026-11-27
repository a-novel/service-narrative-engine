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

func TestRestInstanceList(t *testing.T) {
	t.Parallel()

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodGet, "/v1/instances?projectID="+projectID.String()+"&limit=1&offset=1", nil)
	}

	// paged is the page [validRequest] asks for.
	paged := &core.InstanceListRequest{OwnerID: ownerID, ProjectID: projectID, Limit: 1, Offset: 1}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		// serviceRequest is the request the service receives, [paged] when nil.
		serviceRequest *core.InstanceListRequest
		serviceMock    *serviceMock[*core.InstancePage]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstancePage]{resp: &core.InstancePage{
				Instances: []*core.InstanceDescriptor{coreDescriptor}, Total: 3,
			}},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"instances": []any{jsonDescriptor}, "total": 3.0},
		},
		{
			// A request that names no page leaves the limit and the offset to the service.
			name: "Success/NoPage",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return newRequest(t, http.MethodGet, "/v1/instances?projectID="+projectID.String(), nil)
			},

			serviceRequest: &core.InstanceListRequest{OwnerID: ownerID, ProjectID: projectID},
			serviceMock: &serviceMock[*core.InstancePage]{resp: &core.InstancePage{
				Instances: []*core.InstanceDescriptor{coreDescriptor}, Total: 1,
			}},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"instances": []any{jsonDescriptor}, "total": 1.0},
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

				return newRequest(t, http.MethodGet, "/v1/instances?projectID=nope", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/MalformedLimit",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return newRequest(t, http.MethodGet, "/v1/instances?projectID="+projectID.String()+"&limit=ten", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			// A project without an instance answers an empty list, never null.
			name: "Success/Empty",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstancePage]{resp: &core.InstancePage{
				Instances: []*core.InstanceDescriptor{},
			}},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"instances": []any{}, "total": 0.0},
		},
		{
			// Wiring case: a sentinel only instanceErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests").
			name: "Error/InstanceNotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstancePage]{err: core.ErrInstanceNotFound},

			expectStatus: http.StatusNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestInstanceListService(t)

			if testCase.serviceMock != nil {
				serviceRequest := testCase.serviceRequest
				if serviceRequest == nil {
					serviceRequest = paged
				}

				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestInstanceList(service, configtest.Logger).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
