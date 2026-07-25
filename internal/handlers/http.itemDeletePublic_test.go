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

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	handlersmocks "github.com/a-novel/service-narrative-engine/internal/handlers/mocks"
)

func TestRestItemDeletePublic(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		req  *core.ItemDeleteRequest
		resp *core.Item
		err  error
	}

	testCases := []struct {
		name string

		request *http.Request
		claims  testClaimsState

		serviceMock *serviceMock

		expectStatus   int
		expectResponse any
	}{
		{
			name: "Success",

			request: httptest.NewRequestWithContext(
				t.Context(),
				http.MethodDelete,
				"/item?id=00000000-0000-0000-0000-000000000001",
				nil,
			),

			serviceMock: &serviceMock{
				req: &core.ItemDeleteRequest{
					Actor: testActor,
					ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				resp: &core.Item{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "test item",
					Description: "test description",
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expectStatus: http.StatusOK,
			expectResponse: map[string]any{
				"id":          "00000000-0000-0000-0000-000000000001",
				"name":        "test item",
				"description": "test description",
				"createdAt":   "2021-01-01T00:00:00Z",
				"updatedAt":   "2021-01-01T00:00:00Z",
			},
		},
		{
			name: "Error/AnonymousActor",

			request: httptest.NewRequestWithContext(
				t.Context(), http.MethodDelete, "/item?id=00000000-0000-0000-0000-000000000001", nil,
			),
			claims: testClaimsAnonymous,

			expectStatus: http.StatusForbidden,
		},
		{
			name: "Error/MissingClaims",

			request: httptest.NewRequestWithContext(
				t.Context(), http.MethodDelete, "/item?id=00000000-0000-0000-0000-000000000001", nil,
			),
			claims: testClaimsMissing,

			expectStatus: http.StatusInternalServerError,
		},
		{
			name: "Error/InvalidID",

			request: httptest.NewRequestWithContext(
				t.Context(),
				http.MethodDelete,
				"/item?id=not-a-uuid",
				nil,
			),

			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Error/NotFound",

			request: httptest.NewRequestWithContext(
				t.Context(),
				http.MethodDelete,
				"/item?id=00000000-0000-0000-0000-000000000001",
				nil,
			),

			serviceMock: &serviceMock{
				req: &core.ItemDeleteRequest{
					Actor: testActor,
					ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				err: dao.ErrItemDeleteNotFound,
			},

			expectStatus: http.StatusNotFound,
		},
		{
			name: "Error/Internal",

			request: httptest.NewRequestWithContext(
				t.Context(),
				http.MethodDelete,
				"/item?id=00000000-0000-0000-0000-000000000001",
				nil,
			),

			serviceMock: &serviceMock{
				req: &core.ItemDeleteRequest{
					Actor: testActor,
					ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
				err: errFoo,
			},

			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockItemDeletePublicService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, testCase.serviceMock.req).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewItemDeletePublic(service, config.LoggerDev)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, withTestClaims(testCase.request, testCase.claims))

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
