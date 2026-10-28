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

func TestRestSceneDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	// The request deletes a second scene too, so its id parameter repeats.
	otherScene := *coreScene
	otherScene.ID, otherScene.Position = uuid.MustParse("00000000-0000-0000-0000-000000000012"), 2

	otherJSON := maps.Clone(jsonScene)
	otherJSON["id"], otherJSON["position"] = otherScene.ID.String(), 2.0

	target := "/v1/scenes?projectID=" + projectID.String() + "&id=" + sceneID.String() + "&id=" + otherScene.ID.String()

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
			expectResponse: []any{jsonScene, otherJSON},
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

				return newRequest(t, http.MethodDelete, "/v1/scenes?projectID="+projectID.String()+"&id=not-a-uuid", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/NotFound",

			request: validRequest,

			serviceCalled: true,
			serviceErr:    core.ErrSceneNotFound,

			expectStatus: http.StatusNotFound,
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

			service := handlersmocks.NewMockRestSceneDeleteService(t)

			if testCase.serviceCalled {
				var resp []*core.Scene
				if testCase.serviceErr == nil {
					resp = []*core.Scene{coreScene, &otherScene}
				}

				service.EXPECT().
					Exec(mock.Anything, &core.SceneDeleteRequest{
						OwnerID: ownerID, ProjectID: projectID, IDs: []uuid.UUID{sceneID, otherScene.ID},
					}).
					Return(resp, testCase.serviceErr)
			}

			w := httptest.NewRecorder()
			handlers.NewRestSceneDelete(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
