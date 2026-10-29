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

		serviceMock *serviceMock[[]*core.Layer]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Layer]{resp: []*core.Layer{coreLayer, &otherLayer}},

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
			name: "Error/ProjectNotFound",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Layer]{err: core.ErrProjectNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/LayerNotFound",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Layer]{err: core.ErrLayerNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/RequiredLayer",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Layer]{err: core.ErrRequiredLayer},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/InvalidRequest",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Layer]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Layer]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestLayersDeleteService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.LayersDeleteRequest{
						OwnerID: ownerID, ProjectID: projectID, IDs: []uuid.UUID{layerID, otherLayer.ID},
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestLayersDelete(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			service.AssertExpectations(t)
		})
	}
}
