package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-template/internal/config"
	"github.com/a-novel/service-template/internal/dao"
	"github.com/a-novel/service-template/internal/handlers"
	handlersmocks "github.com/a-novel/service-template/internal/handlers/mocks"
	"github.com/a-novel/service-template/internal/services"
)

func TestRestItemUpdatePublic(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.ItemUpdateRequest
		resp *services.Item
		err  error
	}

	testCases := []struct {
		name string

		request *http.Request

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/item", strings.NewReader(`{
				"id": "00000000-0000-0000-0000-000000000001",
				"name": "updated item",
				"description": "updated description"
			}`)),

			serviceMock: &serviceMock{
				req: &services.ItemUpdateRequest{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "updated item",
					Description: "updated description",
				},
				resp: &services.Item{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "updated item",
					Description: "updated description",
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"id":          "00000000-0000-0000-0000-000000000001",
				"name":        "updated item",
				"description": "updated description",
				"createdAt":   "2021-01-01T00:00:00Z",
				"updatedAt":   "2021-01-02T00:00:00Z",
			},
		},
		{
			name: "Error/InvalidBody",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/item", strings.NewReader(`not json`)),

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/InvalidID",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/item", strings.NewReader(`{
				"id": "not-a-uuid",
				"name": "updated item"
			}`)),

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/EmptyName",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/item", strings.NewReader(`{
				"id": "00000000-0000-0000-0000-000000000001",
				"name": ""
			}`)),

			serviceMock: &serviceMock{
				req: &services.ItemUpdateRequest{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name: "",
				},
				err: services.ErrInvalidRequest,
			},

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/NotFound",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/item", strings.NewReader(`{
				"id": "00000000-0000-0000-0000-000000000001",
				"name": "updated item"
			}`)),

			serviceMock: &serviceMock{
				req: &services.ItemUpdateRequest{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name: "updated item",
				},
				err: dao.ErrItemUpdateNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/Internal",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/item", strings.NewReader(`{
				"id": "00000000-0000-0000-0000-000000000001",
				"name": "updated item"
			}`)),

			serviceMock: &serviceMock{
				req: &services.ItemUpdateRequest{
					ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name: "updated item",
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockItemUpdatePublicService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewItemUpdatePublic(service, config.LoggerDevHttp)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, testCase.request)

			res := w.Result()

			require.Equal(t, testCase.expectStatus, res.StatusCode)

			if testCase.expectResponse != nil {
				data, err := io.ReadAll(res.Body)
				require.NoError(t, errors.Join(err, res.Body.Close()))

				var jsonRes any
				require.NoError(t, json.Unmarshal(data, &jsonRes))
				require.Equal(t, testCase.expectResponse, jsonRes)
			}
		})
	}
}
