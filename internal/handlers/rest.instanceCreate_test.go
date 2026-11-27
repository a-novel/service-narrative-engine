package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	"github.com/a-novel/service-narrative-engine/v2/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/v2/internal/handlers/mocks"
)

func TestRestInstanceCreate(t *testing.T) {
	t.Parallel()

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPost, "/v1/instance", map[string]any{
			"projectID": projectID.String(),
			"plugin":    "scenes-composer",
			"version":   "1.2.0",
			"name":      "Composer",
			"content":   map[string]any{"beats": []any{}},
		})
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[[]*core.InstanceDescriptor]

		expectStatus   int
		expectResponse any
		// expectLocation is the Location header of the response, which names the created instance.
		expectLocation string
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.InstanceDescriptor]{resp: []*core.InstanceDescriptor{coreDescriptor}},

			expectStatus:   http.StatusCreated,
			expectResponse: jsonDescriptor,
			expectLocation: "/v1/instance?projectID=" + projectID.String() + "&id=" + instanceID.String(),
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

				return newRequest(t, http.MethodPost, "/v1/instance", "not an object")
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
			// The batch request names the plugin Instances[0].Plugin; the route names it as its own body does.
			name: "Error/InvalidFields",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.InstanceDescriptor]{err: errors.Join(validator.New().Struct(&struct {
				Instances []struct {
					Plugin string `validate:"required"`
				} `validate:"dive"`
			}{Instances: make([]struct {
				Plugin string `validate:"required"`
			}, 1)}), core.ErrInvalidRequest)},

			expectStatus:   http.StatusUnprocessableEntity,
			expectResponse: invalidFieldsProblem(map[string]any{"plugin": "required"}),
		},
		{
			// The batch operation names the content instances[0].content; the route names it as its own body
			// does.
			name: "Error/InvalidDocument",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.InstanceDescriptor]{
				err: &core.FieldError{Path: "instances[0].content", Rule: "maxdepth"},
			},

			expectStatus:   http.StatusUnprocessableEntity,
			expectResponse: invalidFieldsProblem(map[string]any{"content": "maxdepth"}),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestInstanceCreateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.InstancesCreateRequest{
						OwnerID:   ownerID,
						ProjectID: projectID,
						Instances: []core.InstancesCreateItem{{
							Plugin: "scenes-composer", Version: "1.2.0", Name: "Composer",
							Content: json.RawMessage(`{"beats":[]}`),
						}},
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestInstanceCreate(service, configtest.Logger).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			require.Equal(t, testCase.expectLocation, w.Header().Get("Location"))
		})
	}
}
