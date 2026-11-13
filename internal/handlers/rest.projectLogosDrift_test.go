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

func TestRestProjectLogosDrift(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	removedID := uuid.MustParse("00000000-0000-0000-0000-0000000000ff")

	held := map[string]string{projectID.String(): logoChecksum}

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodPost, "/v1/projects/logos/drift", map[string]any{"logos": held})
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[*core.ProjectLogosDriftResult]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectLogosDriftResult]{
				resp: &core.ProjectLogosDriftResult{
					Logos:   map[uuid.UUID]string{projectID: logoChecksum},
					Removed: []uuid.UUID{removedID},
				},
			},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"logos": map[string]any{projectID.String(): logoChecksum}, "removed": []any{removedID.String()},
			},
		},
		{
			name: "Success/NoDrift",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectLogosDriftResult]{
				resp: &core.ProjectLogosDriftResult{Logos: map[uuid.UUID]string{}, Removed: []uuid.UUID{}},
			},

			expectStatus:   http.StatusOK,
			expectResponse: map[string]any{"logos": map[string]any{}, "removed": []any{}},
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

				return newRequest(t, http.MethodPost, "/v1/projects/logos/drift", "not an object")
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/InvalidRequest",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectLogosDriftResult]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[*core.ProjectLogosDriftResult]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestProjectLogosDriftService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ProjectLogosDriftRequest{OwnerID: ownerID, Logos: held}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestProjectLogosDrift(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			service.AssertExpectations(t)
		})
	}
}
