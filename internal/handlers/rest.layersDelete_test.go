package handlers_test

import (
	"errors"
	"maps"
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

func TestRestLayersDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	// The request deletes a second layer too, so its id parameter repeats.
	otherLayer := *coreLayer
	otherLayer.ID, otherLayer.Position = uuid.MustParse("00000000-0000-0000-0000-000000000022"), 2

	otherJSON := maps.Clone(jsonLayer)
	otherJSON["id"], otherJSON["position"] = otherLayer.ID.String(), 2.0

	target := "/v1/layers?projectID=" + projectID.String() + "&id=" + layerID.String() + "&id=" + otherLayer.ID.String()

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodDelete, target, nil)
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceCalled bool
		serviceErr    error

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceCalled: true,

			expectStatus:   http.StatusOK,
			expectResponse: []any{jsonLayer, otherJSON},
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

				return newRequest(t, http.MethodDelete, "/v1/layers?projectID="+projectID.String()+"&id=not-a-uuid", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/RequiredLayer",

			request: validRequest,

			serviceCalled: true,
			serviceErr:    core.ErrRequiredLayer,

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceCalled: true,
			serviceErr:    errFoo,

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestLayersDeleteService(t)

			if testCase.serviceCalled {
				var resp []*core.Layer
				if testCase.serviceErr == nil {
					resp = []*core.Layer{coreLayer, &otherLayer}
				}

				service.EXPECT().
					Exec(mock.Anything, &core.LayersDeleteRequest{
						OwnerID: ownerID, ProjectID: projectID, IDs: []uuid.UUID{layerID, otherLayer.ID},
					}).
					Return(resp, testCase.serviceErr)
			}

			w := httptest.NewRecorder()
			handlers.NewRestLayersDelete(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
