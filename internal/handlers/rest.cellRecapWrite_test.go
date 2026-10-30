package handlers_test

import (
	"encoding/json"
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

// recapOutput is a consolidate output, as a recap write sends it.
var recapOutput = json.RawMessage(`{"condensed":"Marie.","compact":"Marie.","essentials":"Marie.","names":["Marie"]}`)

func TestRestCellRecapWrite(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPut, "/v1/cell/recap", map[string]any{
			"projectID":     projectID,
			"sceneID":       sceneID,
			"layerID":       layerID,
			"recap":         recapOutput,
			"cellUpdatedAt": updatedAt,
		})
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[struct{}]

		expectStatus int
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[struct{}]{},

			expectStatus: http.StatusNoContent,
		},
		{
			// The owner comes from the token, so naming another one in the body changes nothing.
			name: "Success/OwnerFromToken",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return newRequest(t, http.MethodPut, "/v1/cell/recap", map[string]any{
					"ownerID":       "00000000-0000-0000-0000-0000000000a2",
					"projectID":     projectID,
					"sceneID":       sceneID,
					"layerID":       layerID,
					"recap":         recapOutput,
					"cellUpdatedAt": updatedAt,
				})
			},

			serviceMock: &serviceMock[struct{}]{},

			expectStatus: http.StatusNoContent,
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

				return newRequest(t, http.MethodPut, "/v1/cell/recap", "not an object")
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/InvalidRecap",

			request: validRequest,

			serviceMock: &serviceMock[struct{}]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: validRequest,

			serviceMock: &serviceMock[struct{}]{err: core.ErrProjectNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/CellNotFound",

			request: validRequest,

			serviceMock: &serviceMock[struct{}]{err: core.ErrCellNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/Conflict",

			request: validRequest,

			serviceMock: &serviceMock[struct{}]{err: core.ErrRecapConflict},

			expectStatus: http.StatusConflict,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[struct{}]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestCellRecapWriteService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *core.CellRecapWriteRequest) bool {
						return request.OwnerID == ownerID && request.ProjectID == projectID && request.SceneID == sceneID &&
							request.LayerID == layerID && string(request.Recap) == string(recapOutput) &&
							request.CellUpdatedAt.Equal(updatedAt)
					})).
					Return(testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestCellRecapWrite(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, nil)
			service.AssertExpectations(t)
		})
	}
}
