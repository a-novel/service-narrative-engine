package handlers_test

import (
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

func TestRestCellList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := func(t *testing.T) *http.Request {
		t.Helper()

		target := "/v1/cells?projectID=" + projectID.String() + "&limit=10" +
			"&afterSceneID=" + sceneID.String() + "&afterLayerID=" + layerID.String()

		return newRequest(t, http.MethodGet, target, nil)
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
			expectResponse: map[string]any{"cells": []any{jsonCell}, "total": 3.0},
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

				return newRequest(t, http.MethodGet, "/v1/cells?projectID=not-a-uuid", nil)
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: validRequest,

			serviceCalled: true,
			serviceErr:    core.ErrProjectNotFound,

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/InvalidRequest",

			request: validRequest,

			serviceCalled: true,
			serviceErr:    core.ErrInvalidRequest,

			expectStatus: http.StatusBadRequest,
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

			service := handlersmocks.NewMockRestCellListService(t)

			if testCase.serviceCalled {
				var resp *core.CellPage
				if testCase.serviceErr == nil {
					resp = &core.CellPage{Cells: []*core.Cell{coreCell}, Total: 3}
				}

				service.EXPECT().
					Exec(mock.Anything, &core.CellListRequest{
						OwnerID: ownerID, ProjectID: projectID, Limit: 10, AfterSceneID: sceneID, AfterLayerID: layerID,
					}).
					Return(resp, testCase.serviceErr)
			}

			w := httptest.NewRecorder()
			handlers.NewRestCellList(service, config.LoggerDev).ServeHTTP(w, testCase.request(t))

			requireResponse(t, w.Result(), testCase.expectStatus, testCase.expectResponse)
		})
	}
}
