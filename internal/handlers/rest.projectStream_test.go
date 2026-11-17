package handlers_test

import (
	"errors"
	"iter"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestProjectStream(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		return newRequest(t, http.MethodGet, "/v1/project/stream?id="+projectID.String(), nil)
	}
	resumedAt := func(position string) func(t *testing.T) *http.Request {
		return func(t *testing.T) *http.Request {
			t.Helper()

			request := validRequest(t)
			request.Header.Set("Last-Event-ID", position)

			return request
		}
	}

	at := func(section string, count int) core.ProjectStreamPosition {
		return core.ProjectStreamPosition{Section: section, Count: count}
	}

	testCases := []struct {
		name string

		request func(t *testing.T) *http.Request

		// expectAfter is the position the service resumes after.
		expectAfter *core.ProjectStreamPosition
		serviceMock *serviceMock[iter.Seq2[*core.ProjectRecord, error]]

		expectStatus int
		expectEvents []streamEvent
	}{
		{
			name: "Success",

			request: validRequest,

			serviceMock: &serviceMock[iter.Seq2[*core.ProjectRecord, error]]{resp: sequence([]*core.ProjectRecord{
				{Position: at(core.ProjectStreamProject, 1), Project: coreProject},
				{Position: at(core.ProjectStreamBeats, 1), Beat: coreBeat},
				{Position: at(core.ProjectStreamScenes, 1), Scene: coreScene},
				{Position: at(core.ProjectStreamLayers, 1), Layer: coreLayer},
				{Position: at(core.ProjectStreamCells, 1), Cell: coreCell},
				{Position: at(core.ProjectStreamEnd, 0)},
			}, nil)},

			expectStatus: http.StatusOK,
			expectEvents: []streamEvent{
				{ID: "project:1", Type: "project", Data: jsonProject},
				{ID: "beats:1", Type: "beat", Data: jsonBeat},
				{ID: "scenes:1", Type: "scene", Data: jsonScene},
				{ID: "layers:1", Type: "layer", Data: jsonLayer},
				{ID: "cells:1", Type: "cell", Data: jsonCell},
				{ID: "end", Type: "end", Data: map[string]any{}},
			},
		},
		{
			name: "Success/Resume",

			request: resumedAt("scenes:41"),

			expectAfter: &core.ProjectStreamPosition{Section: core.ProjectStreamScenes, Count: 41},
			serviceMock: &serviceMock[iter.Seq2[*core.ProjectRecord, error]]{resp: sequence([]*core.ProjectRecord{
				{Position: at(core.ProjectStreamEnd, 0)},
			}, nil)},

			expectStatus: http.StatusOK,
			expectEvents: []streamEvent{{ID: "end", Type: "end", Data: map[string]any{}}},
		},
		{
			// The client resumes the cut stream.
			name: "Success/Cut",

			request: validRequest,

			serviceMock: &serviceMock[iter.Seq2[*core.ProjectRecord, error]]{resp: sequence([]*core.ProjectRecord{
				{Position: at(core.ProjectStreamProject, 1), Project: coreProject},
			}, errFoo)},

			expectStatus: http.StatusOK,
			expectEvents: []streamEvent{{ID: "project:1", Type: "project", Data: jsonProject}},
		},
		{
			name: "Success/Ended",

			request: resumedAt("end"),

			expectAfter: &core.ProjectStreamPosition{Section: core.ProjectStreamEnd},
			serviceMock: &serviceMock[iter.Seq2[*core.ProjectRecord, error]]{err: core.ErrStreamEnded},

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

				return newRequest(t, http.MethodGet, "/v1/project/stream?id=not-a-uuid", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/PositionWithoutCount",

			request: resumedAt("scenes"),

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/PositionWithMalformedCount",

			request: resumedAt("scenes:many"),

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/InvalidPosition",

			request: resumedAt("chapters:2"),

			expectAfter: &core.ProjectStreamPosition{Section: "chapters", Count: 2},
			serviceMock: &serviceMock[iter.Seq2[*core.ProjectRecord, error]]{err: core.ErrInvalidRequest},

			expectStatus: http.StatusUnprocessableEntity,
		},
		{
			// Wiring case: a sentinel only composerErrors holds, to prove the handler uses that map (see
			// CONTRIBUTING, "Handler tests"), whether or not the route produces it.
			name: "Error/ProjectFull",

			request: validRequest,

			serviceMock: &serviceMock[iter.Seq2[*core.ProjectRecord, error]]{err: core.ErrProjectFull},

			expectStatus: http.StatusConflict,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockRestProjectStreamService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ProjectStreamRequest{
						OwnerID: ownerID, ProjectID: projectID, After: testCase.expectAfter,
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			w := httptest.NewRecorder()
			handlers.NewRestProjectStream(service, configtest.Logger).ServeHTTP(w, testCase.request(t))

			if testCase.expectEvents != nil {
				requireEvents(t, w.Result(), testCase.expectEvents)
			} else {
				require.Equal(t, testCase.expectStatus, w.Code)
			}
		})
	}

	// A client the handler cannot write to ends the stream, and the handler reads no further record.
	clientCases := []struct {
		name string

		// client wraps the recorder as the writer the handler answers.
		client func(w *httptest.ResponseRecorder) http.ResponseWriter

		// expectPulled is how many records the handler reads from the service.
		expectPulled int
	}{
		{
			// Every write fails once the client left: the first event fails to send.
			name: "Success/ClientLeft",

			client: func(w *httptest.ResponseRecorder) http.ResponseWriter { return leftClient{w} },

			expectPulled: 1,
		},
		{
			// A writer that cannot flush cannot open the stream.
			name: "Error/CannotFlush",

			client: func(w *httptest.ResponseRecorder) http.ResponseWriter { return struct{ http.ResponseWriter }{w} },

			expectPulled: 0,
		},
	}

	for _, testCase := range clientCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var pulled int

			records := func(yield func(*core.ProjectRecord, error) bool) {
				for _, record := range []*core.ProjectRecord{
					{Position: at(core.ProjectStreamProject, 1), Project: coreProject},
					{Position: at(core.ProjectStreamEnd, 0)},
				} {
					pulled++

					if !yield(record, nil) {
						return
					}
				}
			}

			service := handlersmocks.NewMockRestProjectStreamService(t)
			service.EXPECT().
				Exec(mock.Anything, &core.ProjectStreamRequest{OwnerID: ownerID, ProjectID: projectID}).
				Return(records, nil)

			w := testCase.client(httptest.NewRecorder())
			handlers.NewRestProjectStream(service, configtest.Logger).ServeHTTP(w, validRequest(t))
			require.Equal(t, testCase.expectPulled, pulled)
		})
	}
}
