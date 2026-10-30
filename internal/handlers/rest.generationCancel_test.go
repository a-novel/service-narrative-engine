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

func TestRestGenerationCancel(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodDelete, "/v1/generation?id="+genaiID.String(), nil)
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		serviceMock *serviceMock[*core.GenerationResult]

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[*core.GenerationResult]{resp: &core.GenerationResult{
				ID: genaiID, Status: core.GenerationStatusSucceeded, Output: json.RawMessage(suggestOutput),
				Model: suggestModel,
			}},

			expectStatus:   http.StatusOK,
			expectResponse: jsonResult,
		},
		{
			name: "Success/Failed",

			request: validRequest,

			serviceMock: &serviceMock[*core.GenerationResult]{resp: &core.GenerationResult{
				ID: genaiID, Status: core.GenerationStatusFailed, Error: "incomplete output: max_output_tokens",
			}},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"id": genaiID.String(), "status": "failed", "error": "incomplete output: max_output_tokens",
			},
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
			name: "Error/MalformedID",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return newRequest(t, http.MethodDelete, "/v1/generation?id=not-a-uuid", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/NotFound",

			request: validRequest,

			serviceMock: &serviceMock[*core.GenerationResult]{err: core.ErrGenerationNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/GenaiUnavailable",

			request: validRequest,

			serviceMock: &serviceMock[*core.GenerationResult]{err: core.ErrGenaiUnavailable},

			expectStatus: http.StatusServiceUnavailable,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[*core.GenerationResult]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestGenerationCancelService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.GenerationCancelRequest{OwnerID: ownerID, ID: genaiID}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestGenerationCancel(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
			service.AssertExpectations(t)
		})
	}
}
