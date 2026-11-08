package handlers_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestProjectPictureGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	target := "/v1/project/picture?projectID=" + projectID.String() + "&picture=logo"
	picture := &core.ProjectPicture{MediaType: "image/webp", Data: []byte("picture"), Version: logoVersion}
	etag := `"1609661730123456"`

	testCases := []struct {
		name string

		target      string
		caller      *uuid.UUID
		ifNoneMatch string

		serviceMock *serviceMock[*core.ProjectPicture]

		expectStatus int
		expectBody   string
	}{
		{
			name: "Success",

			target: target,
			caller: &ownerID,

			serviceMock: &serviceMock[*core.ProjectPicture]{resp: picture},

			expectStatus: http.StatusOK,
			expectBody:   "picture",
		},
		{
			name: "Success/NotModified",

			target:      target,
			caller:      &ownerID,
			ifNoneMatch: etag,

			serviceMock: &serviceMock[*core.ProjectPicture]{resp: picture},

			expectStatus: http.StatusNotModified,
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
			name: "Error/InvalidRequest",

			target: target,
			caller: &ownerID,

			serviceMock: &serviceMock[*core.ProjectPicture]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/ProjectNotFound",

			target: target,
			caller: &ownerID,

			serviceMock: &serviceMock[*core.ProjectPicture]{err: core.ErrProjectNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/PictureNotFound",

			target: target,
			caller: &ownerID,

			serviceMock: &serviceMock[*core.ProjectPicture]{err: core.ErrPictureNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/Internal",

			target: target,
			caller: &ownerID,

			serviceMock: &serviceMock[*core.ProjectPicture]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestProjectPictureGetService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ProjectPictureGetRequest{
						OwnerID: ownerID, ProjectID: projectID, Picture: core.ProjectPictureLogo,
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			request := asUser(newRequest(t, http.MethodGet, testCase.target, nil), testCase.caller)
			if testCase.ifNoneMatch != "" {
				request.Header.Set("If-None-Match", testCase.ifNoneMatch)
			}

			w := httptest.NewRecorder()
			handlers.NewRestProjectPictureGet(service, config.LoggerDev).ServeHTTP(w, request)

			response := w.Result()
			require.Equal(t, testCase.expectStatus, response.StatusCode)

			body, err := io.ReadAll(response.Body)
			require.NoError(t, errors.Join(err, response.Body.Close()))

			if testCase.expectStatus >= http.StatusBadRequest {
				return
			}

			require.Equal(t, testCase.expectBody, string(body))
			require.Equal(t, etag, response.Header.Get("ETag"))
			require.Equal(t, "no-cache", response.Header.Get("Cache-Control"))

			// A 304 leaves out the headers that describe the body.
			if testCase.expectStatus == http.StatusOK {
				require.Equal(t, "image/webp", response.Header.Get("Content-Type"))
				require.Equal(t, "nosniff", response.Header.Get("X-Content-Type-Options"))
				require.Equal(t, "Sun, 03 Jan 2021 08:15:30 GMT", response.Header.Get("Last-Modified"))
			}
		})
	}
}
