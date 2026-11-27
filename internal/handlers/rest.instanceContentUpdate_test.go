package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	"github.com/a-novel/service-narrative-engine/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/v2/internal/handlers/mocks"
)

// TestRestInstanceContentUpdate also covers instanceErrors, the error map the instance, default instance and
// links handlers share: each entry of it has its case here (see CONTRIBUTING, "Handler tests").
func TestRestInstanceContentUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPut, "/v1/instance/content", map[string]any{
			"projectID": projectID.String(),
			"id":        instanceID.String(),
			"version":   "1.2.0",
			"content":   map[string]any{"beats": []any{}},
			"updatedAt": "2021-01-03T08:15:30.123456Z",
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
			// The request's updatedAt reaches the service with its microseconds.
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

				return newRequest(t, http.MethodPut, "/v1/instance/content", "not an object")
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			// An instant that is no RFC 3339 text fails the decoding.
			name: "Error/MalformedUpdatedAt",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return newRequest(t, http.MethodPut, "/v1/instance/content", map[string]any{
					"projectID": projectID.String(), "id": instanceID.String(), "version": "1.2.0",
					"content": map[string]any{}, "updatedAt": 1609747200,
				})
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/InvalidRequest",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceDescriptor]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			// A document over a cap names its field and the rule it broke.
			name: "Error/InvalidField",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceDescriptor]{
				err: &core.FieldError{Path: "content", Rule: "maxbytes"},
			},

			expectStatus:   http.StatusUnprocessableEntity,
			expectResponse: invalidFieldsProblem(map[string]any{"content": "maxbytes"}),
		},
		{
			name: "Error/ProjectNotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceDescriptor]{err: core.ErrProjectNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/InstanceNotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceDescriptor]{err: core.ErrInstanceNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/InstanceStale",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceDescriptor]{err: core.ErrInstanceStale},

			expectStatus: http.StatusConflict,
		},
		{
			name: "Error/InstanceDefault",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceDescriptor]{err: core.ErrInstanceDefault},

			expectStatus: http.StatusConflict,
		},
		{
			name: "Error/InstancesFull",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceDescriptor]{err: core.ErrInstancesFull},

			expectStatus:   http.StatusConflict,
			expectResponse: limitProblem("instances"),
		},
		{
			name: "Error/ProjectLinksStale",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceDescriptor]{err: core.ErrProjectLinksStale},

			expectStatus: http.StatusConflict,
		},
		{
			name: "Error/ProjectBudget",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceDescriptor]{err: core.ErrProjectBudget},

			expectStatus:   http.StatusConflict,
			expectResponse: limitProblem("projectBytes"),
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[*core.InstanceDescriptor]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestInstanceContentUpdateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.InstanceContentUpdateRequest{
						OwnerID:   ownerID,
						ProjectID: projectID,
						ID:        instanceID,
						Version:   "1.2.0",
						Content:   json.RawMessage(`{"beats":[]}`),
						UpdatedAt: preciseAt,
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestInstanceContentUpdate(service, configtest.Logger).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
