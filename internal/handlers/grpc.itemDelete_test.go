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

	"github.com/a-novel/service-template/internal/dao"
	"github.com/a-novel/service-template/internal/handlers"
	handlersmocks "github.com/a-novel/service-template/internal/handlers/mocks"
	"github.com/a-novel/service-template/internal/handlers/protogen"
	"github.com/a-novel/service-template/internal/services"
)

func TestGrpcItemDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		resp *services.Item
		err  error
	}

	testCases := []struct {
		name string

		request *protogen.ItemDeleteRequest

		serviceMock *serviceMock

		expect       *protogen.ItemDeleteResponse
		expectStatus codes.Code
	}{
		{
			name: "Success",

			request: &protogen.ItemDeleteRequest{
				Id: "00000000-0000-0000-0000-000000000001",
			},

			serviceMock: &serviceMock{
				resp: &services.Item{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "test item",
					Description: "test description",
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expectStatus: codes.OK,
			expect: &protogen.ItemDeleteResponse{
				Item: &protogen.Item{
					Id:          "00000000-0000-0000-0000-000000000001",
					Name:        "test item",
					Description: "test description",
					CreatedAt:   "2021-01-01T00:00:00Z",
					UpdatedAt:   "2021-01-01T00:00:00Z",
				},
			},
		},
		{
			name: "Error/InvalidID",

			request: &protogen.ItemDeleteRequest{
				Id: "not-a-uuid",
			},

			expectStatus: codes.InvalidArgument,
		},
		{
			name: "Error/NotFound",

			request: &protogen.ItemDeleteRequest{
				Id: "00000000-0000-0000-0000-000000000001",
			},

			serviceMock: &serviceMock{
				err: dao.ErrItemDeleteNotFound,
			},

			expectStatus: codes.NotFound,
		},
		{
			name: "Error/Internal",

			request: &protogen.ItemDeleteRequest{
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

			service := handlersmocks.NewMockItemDeleteService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &services.ItemDeleteRequest{
						ID: uuid.MustParse(testCase.request.GetId()),
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewItemDelete(service)

			res, err := handler.ItemDelete(t.Context(), testCase.request)
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
