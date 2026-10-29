package handlers_test

import (
	"errors"
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

func TestRestBeatRecapWrite(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	// The second scene holds no manuscript, so its source carries no stamp.
	otherScene := uuid.MustParse("00000000-0000-0000-0000-000000000012")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPut, "/v1/beat/recap", map[string]any{
			"projectID":     projectID,
			"beatID":        beatID,
			"recap":         recapOutput,
			"beatUpdatedAt": updatedAt,
			"sources": []any{
				map[string]any{"sceneID": sceneID, "cellUpdatedAt": updatedAt},
				map[string]any{"sceneID": otherScene, "cellUpdatedAt": nil},
			},
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

				return newRequest(t, http.MethodPut, "/v1/beat/recap", "not an object")
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: validRequest,

			serviceMock: &serviceMock[struct{}]{err: core.ErrProjectNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/BeatNotFound",

			request: validRequest,

			serviceMock: &serviceMock[struct{}]{err: core.ErrBeatNotFound},

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

			service := handlersmocks.NewMockRestBeatRecapWriteService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *core.BeatRecapWriteRequest) bool {
						return request.OwnerID == ownerID && request.ProjectID == projectID && request.BeatID == beatID &&
							string(request.Recap) == string(recapOutput) && request.BeatUpdatedAt.Equal(updatedAt) &&
							len(request.Sources) == 2 && request.Sources[0].SceneID == sceneID &&
							request.Sources[0].CellUpdatedAt.Equal(updatedAt) && request.Sources[1].SceneID == otherScene &&
							request.Sources[1].CellUpdatedAt.IsZero()
					})).
					Return(testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestBeatRecapWrite(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, nil)
			service.AssertExpectations(t)
		})
	}
}
