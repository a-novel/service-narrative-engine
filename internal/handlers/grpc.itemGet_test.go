package handlers_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel/service-template/internal/core"
	"github.com/a-novel/service-template/internal/handlers"
	handlersmocks "github.com/a-novel/service-template/internal/handlers/mocks"
	templatev0 "github.com/a-novel/service-template/internal/handlers/protogen/anovel/template/v0"
)

func TestGrpcItemGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		resp *core.Item
		err  error
	}

	testCases := []struct {
		name string

		request *templatev0.ItemGetRequest

		serviceMock *serviceMock

		expect       *templatev0.ItemGetResponse
		expectStatus codes.Code
	}{
		{
			name: "Success",

			request: &templatev0.ItemGetRequest{
				Id: "00000000-0000-0000-0000-000000000001",
			},

			serviceMock: &serviceMock{
				resp: &core.Item{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "test item",
					Description: "test description",
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 123456000, time.UTC),
					UpdatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expectStatus: codes.OK,
			expect: &templatev0.ItemGetResponse{
				Item: &templatev0.Item{
					Id:          "00000000-0000-0000-0000-000000000001",
					Name:        "test item",
					Description: "test description",
					CreatedAt:   "2021-01-01T00:00:00.123456Z",
					UpdatedAt:   "2021-01-01T00:00:00Z",
				},
			},
		},
		{
			name: "Error/InvalidID",

			request: &templatev0.ItemGetRequest{
				Id: "not-a-uuid",
			},

			expectStatus: codes.InvalidArgument,
		},
		{
			name: "Error/NotFound",

			request: &templatev0.ItemGetRequest{
				Id: "00000000-0000-0000-0000-000000000001",
			},

			serviceMock: &serviceMock{
				err: core.ErrItemGetNotFound,
			},

			expectStatus: codes.NotFound,
		},
		{
			name: "Error/Internal",

			request: &templatev0.ItemGetRequest{
				Id: "00000000-0000-0000-0000-000000000001",
			},

			serviceMock: &serviceMock{
				err: errFoo,
			},

			expectStatus: codes.Internal,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := handlersmocks.NewMockGrpcItemGetService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &core.ItemGetRequest{
						ID: uuid.MustParse(testCase.request.GetId()),
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewGrpcItemGet(service)

			res, err := handler.ItemGet(t.Context(), testCase.request)
			resSt, ok := status.FromError(err)
			require.True(t, ok, resSt.Code().String())
			require.Equal(
				t,
				testCase.expectStatus, resSt.Code(),
				"expected status code %s, got %s (%v)", testCase.expectStatus, resSt.Code(), err,
			)
			require.Equal(t, testCase.expect, res)

			service.AssertExpectations(t)
		})
	}
}
