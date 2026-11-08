package handlers_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/a-novel-kit/golib/picture"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestProjectPictureUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	data := []byte("picture")
	target := "/v1/project/picture?projectID=" + projectID.String() + "&picture=logo"

	// refusedResponse is the problem a refused picture answers, naming the rule it broke.
	refusedResponse := func(reason string) map[string]any {
		return map[string]any{
			"type":   "about:blank",
			"title":  "Unprocessable Entity",
			"status": float64(http.StatusUnprocessableEntity),
			"tags":   map[string]any{"reason": reason},
		}
	}

	testCases := []struct {
		name string

		target string
		caller *uuid.UUID
		// bodyLimit stands for the router's body limit, when set.
		bodyLimit int64

		serviceMock *serviceMock[*core.Project]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			target: target,
			caller: &ownerID,

			serviceMock: &serviceMock[*core.Project]{resp: coreProject},

			expectStatus:   http.StatusOK,
			expectResponse: jsonProject,
		},
		{
			name: "Error/Anonymous",

			target: target,

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/MalformedProjectID",

			target: "/v1/project/picture?projectID=not-a-uuid&picture=logo",
			caller: &ownerID,

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/TooLarge",

			target:    target,
			caller:    &ownerID,
			bodyLimit: int64(len(data)) - 1,

			expectStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name: "Error/InvalidRequest",

			target: target,
			caller: &ownerID,

			serviceMock: &serviceMock[*core.Project]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/ProjectNotFound",

			target: target,
			caller: &ownerID,

			serviceMock: &serviceMock[*core.Project]{err: core.ErrProjectNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/Format",

			target: target,
			caller: &ownerID,

			serviceMock: &serviceMock[*core.Project]{err: picture.ErrFormat},

			expectStatus:   http.StatusUnprocessableEntity,
			expectResponse: refusedResponse("format"),
		},
		{
			name: "Error/Ratio",

			target: target,
			caller: &ownerID,

			serviceMock: &serviceMock[*core.Project]{err: picture.ErrRatio},

			expectStatus:   http.StatusUnprocessableEntity,
			expectResponse: refusedResponse("ratio"),
		},
		{
			name: "Error/Dimensions",

			target: target,
			caller: &ownerID,

			serviceMock: &serviceMock[*core.Project]{err: picture.ErrDimensions},

			expectStatus:   http.StatusUnprocessableEntity,
			expectResponse: refusedResponse("dimensions"),
		},
		{
			name: "Error/Internal",

			target: target,
			caller: &ownerID,

			serviceMock: &serviceMock[*core.Project]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestProjectPictureUpdateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ProjectPictureUpdateRequest{
						OwnerID: ownerID, ProjectID: projectID, Picture: core.ProjectPictureLogo, Data: data,
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()

			var body io.Reader = bytes.NewReader(data)
			if testCase.bodyLimit > 0 {
				body = http.MaxBytesReader(w, io.NopCloser(body), testCase.bodyLimit)
			}

			request := httptest.NewRequestWithContext(t.Context(), http.MethodPut, testCase.target, body)
			handlers.NewRestProjectPictureUpdate(service, config.LoggerDev).ServeHTTP(w, asUser(request, testCase.caller))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			service.AssertExpectations(t)
		})
	}
}
