package handlers_test

import (
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestGenerationStream(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodGet, "/v1/generation/stream?id="+genaiID.String(), nil)
	}
	resumed := func(t *testing.T) *http.Request {
		t.Helper()

		request := validRequest(t)
		request.Header.Set("Last-Event-ID", "4")

		return request
	}

	succeeded := &core.GenerationResult{
		ID: genaiID, Status: core.GenerationStatusSucceeded, Output: json.RawMessage(suggestOutput),
		Model: suggestModel,
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		// expectAfter is the position the service resumes after.
		expectAfter *int64
		serviceMock *serviceMock[iter.Seq2[*core.GenerationEvent, error]]

		expectStatus   int
		expectEvents   []streamEvent
		expectLocation string
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[iter.Seq2[*core.GenerationEvent, error]]{resp: sequence([]*core.GenerationEvent{
				{Sequence: 1, Delta: "La "},
				{Sequence: 2, Delta: "lettre"},
				{Sequence: 3, Result: succeeded},
			}, nil)},

			expectStatus: http.StatusOK,
			expectEvents: []streamEvent{
				{ID: "1", Type: "delta", Data: map[string]any{"text": "La "}},
				{ID: "2", Type: "delta", Data: map[string]any{"text": "lettre"}},
				{ID: "3", Type: "result", Data: jsonResult},
			},
		},
		{
			name: "Success/Resume",

			request: resumed,

			expectAfter: lo.ToPtr[int64](4),
			serviceMock: &serviceMock[iter.Seq2[*core.GenerationEvent, error]]{resp: sequence([]*core.GenerationEvent{
				{Sequence: 5, Result: succeeded},
			}, nil)},

			expectStatus: http.StatusOK,
			expectEvents: []streamEvent{{ID: "5", Type: "result", Data: jsonResult}},
		},
		{
			// The client resumes the cut stream.
			name: "Success/Cut",

			request: validRequest,

			serviceMock: &serviceMock[iter.Seq2[*core.GenerationEvent, error]]{resp: sequence([]*core.GenerationEvent{
				{Sequence: 1, Delta: "La "},
			}, core.ErrGenaiUnavailable)},

			expectStatus: http.StatusOK,
			expectEvents: []streamEvent{{ID: "1", Type: "delta", Data: map[string]any{"text": "La "}}},
		},
		{
			name: "Success/NotStreamed",

			request: validRequest,

			serviceMock: &serviceMock[iter.Seq2[*core.GenerationEvent, error]]{err: core.ErrGenerationNotStreamed},

			expectStatus:   http.StatusSeeOther,
			expectLocation: "/v1/generation?id=" + genaiID.String(),
		},
		{
			name: "Success/Ended",

			request: resumed,

			expectAfter: lo.ToPtr[int64](4),
			serviceMock: &serviceMock[iter.Seq2[*core.GenerationEvent, error]]{err: core.ErrStreamEnded},

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
			name: "Error/MalformedID",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				return newRequest(t, http.MethodGet, "/v1/generation/stream?id=not-a-uuid", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/MalformedLastEventID",

			request: func(t *testing.T) *http.Request {
				t.Helper()

				request := validRequest(t)
				request.Header.Set("Last-Event-ID", "result")

				return request
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/NotFound",

			request: validRequest,

			serviceMock: &serviceMock[iter.Seq2[*core.GenerationEvent, error]]{err: core.ErrGenerationNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/GenaiUnavailable",

			request: validRequest,

			serviceMock: &serviceMock[iter.Seq2[*core.GenerationEvent, error]]{err: core.ErrGenaiUnavailable},

			expectStatus: http.StatusServiceUnavailable,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[iter.Seq2[*core.GenerationEvent, error]]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestGenerationStreamService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.GenerationStreamRequest{OwnerID: ownerID, ID: genaiID, After: testCase.expectAfter}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestGenerationStream(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			if testCase.expectEvents != nil {
				requireEvents(t, w.Result(), testCase.expectEvents)
			} else {
				require.Equal(t, testCase.expectStatus, w.Code)
				require.Equal(t, testCase.expectLocation, w.Header().Get("Location"))
			}

			service.AssertExpectations(t)
		})
	}
}
