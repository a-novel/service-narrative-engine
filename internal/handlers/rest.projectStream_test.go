package handlers_test

import (
	"errors"
	"iter"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config"
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

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/NotFound",

			request: validRequest,

			serviceMock: &serviceMock[iter.Seq2[*core.ProjectRecord, error]]{err: core.ErrProjectNotFound},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			serviceMock: &serviceMock[iter.Seq2[*core.ProjectRecord, error]]{err: errFoo},

			expectStatus: http.StatusInternalServerError,
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
			handlers.NewRestProjectStream(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			if testCase.expectEvents != nil {
				requireEvents(t, w.Result(), testCase.expectEvents)
			} else {
				require.Equal(t, testCase.expectStatus, w.Code)
			}

			service.AssertExpectations(t)
		})
	}
}
