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

func TestRestScenesDelete(t *testing.T) {
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

		serviceMock *serviceMock[[]*core.Scene]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Scene]{resp: []*core.Scene{coreScene, &otherScene}},

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
			name: "Error/ProjectNotFound",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Scene]{err: core.ErrProjectNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/SceneNotFound",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Scene]{err: core.ErrSceneNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/InvalidRequest",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Scene]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[[]*core.Scene]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestScenesDeleteService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ScenesDeleteRequest{
						OwnerID: ownerID, ProjectID: projectID, IDs: []uuid.UUID{sceneID, otherScene.ID},
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestScenesDelete(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			service.AssertExpectations(t)
		})
	}
}
