package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestCellWrite(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPut, "/v1/cell", map[string]any{
			"projectID": projectID,
			"sceneID":   sceneID,
			"layerID":   layerID,
			"content":   coreCell.Content,
		})
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[*core.WrittenCell]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[*core.WrittenCell]{resp: coreWrittenCell},

			expectStatus:   http.StatusOK,
			expectResponse: jsonWrittenCell,
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

				return newRequest(t, http.MethodPut, "/v1/cell", "not an object")
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.WrittenCell]{err: core.ErrProjectNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/SceneNotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.WrittenCell]{err: core.ErrSceneNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/LayerNotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.WrittenCell]{err: core.ErrLayerNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/SceneContentFull",

			request: validRequest,

			serviceMock: &serviceMock[*core.WrittenCell]{err: core.ErrProjectFull},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Error/InvalidContent",

			request: validRequest,

			serviceMock: &serviceMock[*core.WrittenCell]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[*core.WrittenCell]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestCellWriteService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.CellWriteRequest{
						OwnerID:   ownerID,
						ProjectID: projectID,
						SceneID:   sceneID,
						LayerID:   layerID,
						Content:   coreCell.Content,
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestCellWrite(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			service.AssertExpectations(t)
		})
	}
}
