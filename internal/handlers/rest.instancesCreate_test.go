package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	"github.com/a-novel/service-narrative-engine/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/v2/internal/handlers/mocks"
)

func TestRestInstancesCreate(t *testing.T) {
	t.Parallel()

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPost, "/v1/instances", map[string]any{
			"projectID": projectID.String(),
			"instances": []map[string]any{
				{"plugin": "scenes-composer", "version": "1.2.0", "name": "Composer", "content": map[string]any{}},
				{"plugin": "characters", "version": "0.1.0", "name": "Characters", "content": []any{}},
			},
		})
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[[]*core.InstanceDescriptor]

		expectStatus   int
		expectResponse any
		// expectLocation is the Location header of the response, which names the list of the instances.
		expectLocation string
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.InstanceDescriptor]{
				resp: []*core.InstanceDescriptor{coreDescriptor, coreDescriptor},
			},

			expectStatus:   http.StatusCreated,
			expectResponse: []any{jsonDescriptor, jsonDescriptor},
			expectLocation: "/v1/instances?projectID=" + projectID.String(),
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

				return newRequest(t, http.MethodPost, "/v1/instances", "not an object")
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			// Wiring case: a sentinel only instanceErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests").
			name: "Error/InstancesFull",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.InstanceDescriptor]{err: core.ErrInstancesFull},

			expectStatus:   http.StatusConflict,
			expectResponse: limitProblem("instances"),
		},
		{
			// A document over a cap keeps the path the batch gives it.
			name: "Error/InvalidDocument",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.InstanceDescriptor]{
				err: &core.FieldError{Path: "instances[1].content", Rule: "maxbytes"},
			},

			expectStatus:   http.StatusUnprocessableEntity,
			expectResponse: invalidFieldsProblem(map[string]any{"instances[1].content": "maxbytes"}),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestInstancesCreateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.InstancesCreateRequest{
						OwnerID:   ownerID,
						ProjectID: projectID,
						Instances: []core.InstancesCreateItem{
							{
								Plugin: "scenes-composer", Version: "1.2.0", Name: "Composer",
								Content: json.RawMessage(`{}`),
							},
							{Plugin: "characters", Version: "0.1.0", Name: "Characters", Content: json.RawMessage(`[]`)},
						},
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestInstancesCreate(service, configtest.Logger).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			require.Equal(t, testCase.expectLocation, w.Header().Get("Location"))
		})
	}
}
