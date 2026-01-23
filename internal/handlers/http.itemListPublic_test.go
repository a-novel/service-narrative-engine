package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-template/internal/config"
	"github.com/a-novel/service-template/internal/handlers"
	handlersmocks "github.com/a-novel/service-template/internal/handlers/mocks"
	"github.com/a-novel/service-template/internal/services"
)

func TestRestItemListPublic(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *services.ItemListRequest
		resp []*services.Item
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

			request: httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items?limit=10&offset=0", nil),

			serviceMock: &serviceMock{
				req: &services.ItemListRequest{Limit: 10, Offset: 0},
				resp: []*services.Item{
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						Name:      "item two",
						CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						Name:      "item one",
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expectStatus: http.StatusOK,
			expectResponse: []any{
				map[string]any{
					"id":        "00000000-0000-0000-0000-000000000002",
					"name":      "item two",
					"createdAt": "2021-01-02T00:00:00Z",
					"updatedAt": "2021-01-02T00:00:00Z",
				},
				map[string]any{
					"id":        "00000000-0000-0000-0000-000000000001",
					"name":      "item one",
					"createdAt": "2021-01-01T00:00:00Z",
					"updatedAt": "2021-01-01T00:00:00Z",
				},
			},
		},
		{
			name: "Error/Internal",

			request: httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items?limit=10&offset=0", nil),

			serviceMock: &serviceMock{
				req: &services.ItemListRequest{Limit: 10, Offset: 0},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockItemListPublicService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewItemListPublic(service, config.LoggerDevHttp)
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
