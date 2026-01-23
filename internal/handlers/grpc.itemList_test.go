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

	"github.com/a-novel/service-template/internal/handlers"
	handlersmocks "github.com/a-novel/service-template/internal/handlers/mocks"
	"github.com/a-novel/service-template/internal/handlers/protogen"
	"github.com/a-novel/service-template/internal/services"
)

func TestGrpcItemList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		resp []*services.Item
		err  error
	}

	testCases := []struct {
		name string

		request *protogen.ItemListRequest

		serviceMock *serviceMock

		expect       *protogen.ItemListResponse
		expectStatus codes.Code
	}{
		{
			name: "Success",

			request: &protogen.ItemListRequest{
				Limit:  10,
				Offset: 0,
			},

			serviceMock: &serviceMock{
				resp: []*services.Item{
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						Name:      "item one",
						CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						Name:      "item two",
						CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expectStatus: codes.OK,
			expect: &protogen.ItemListResponse{
				Items: []*protogen.Item{
					{
						Id:        "00000000-0000-0000-0000-000000000001",
						Name:      "item one",
						CreatedAt: "2021-01-03T00:00:00Z",
						UpdatedAt: "2021-01-03T00:00:00Z",
					},
					{
						Id:        "00000000-0000-0000-0000-000000000002",
						Name:      "item two",
						CreatedAt: "2021-01-02T00:00:00Z",
						UpdatedAt: "2021-01-02T00:00:00Z",
					},
				},
			},
		},
		{
			name: "Error/InvalidArgument",

			request: &protogen.ItemListRequest{
				Limit:  0,
				Offset: 0,
			},

			serviceMock: &serviceMock{
				err: services.ErrInvalidRequest,
			},

			expectStatus: codes.InvalidArgument,
		},
		{
			name: "Error/Internal",

			request: &protogen.ItemListRequest{
				Limit:  10,
				Offset: 0,
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

			service := handlersmocks.NewMockItemListService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &services.ItemListRequest{
						Limit:  int(testCase.request.GetLimit()),
						Offset: int(testCase.request.GetOffset()),
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewItemList(service)

			res, err := handler.ItemList(t.Context(), testCase.request)
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
