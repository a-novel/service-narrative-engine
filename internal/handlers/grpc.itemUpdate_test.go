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

func TestGrpcItemUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		resp *services.Item
		err  error
	}

	testCases := []struct {
		name string

		request *protogen.ItemUpdateRequest

		serviceMock *serviceMock

		expect       *protogen.ItemUpdateResponse
		expectStatus codes.Code
	}{
		{
			name: "Success",

			request: &protogen.ItemUpdateRequest{
				Id:          "00000000-0000-0000-0000-000000000001",
				Name:        "updated item",
				Description: "updated description",
			},

			serviceMock: &serviceMock{
				resp: &services.Item{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "updated item",
					Description: "updated description",
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			expectStatus: codes.OK,
			expect: &protogen.ItemUpdateResponse{
				Item: &protogen.Item{
					Id:          "00000000-0000-0000-0000-000000000001",
					Name:        "updated item",
					Description: "updated description",
					CreatedAt:   "2021-01-01T00:00:00Z",
					UpdatedAt:   "2021-01-02T00:00:00Z",
				},
			},
		},
		{
			name: "Error/InvalidID",

			request: &protogen.ItemUpdateRequest{
				Id:   "not-a-uuid",
				Name: "updated item",
			},

			expectStatus: codes.InvalidArgument,
		},
		{
			name: "Error/InvalidArgument",

			request: &protogen.ItemUpdateRequest{
				Id:   "00000000-0000-0000-0000-000000000001",
				Name: "",
			},

			serviceMock: &serviceMock{
				err: services.ErrInvalidRequest,
			},

			expectStatus: codes.InvalidArgument,
		},
		{
			name: "Error/NotFound",

			request: &protogen.ItemUpdateRequest{
				Id:   "00000000-0000-0000-0000-000000000001",
				Name: "updated item",
			},

			serviceMock: &serviceMock{
				err: dao.ErrItemUpdateNotFound,
			},

			expectStatus: codes.NotFound,
		},
		{
			name: "Error/Internal",

			request: &protogen.ItemUpdateRequest{
				Id:   "00000000-0000-0000-0000-000000000001",
				Name: "updated item",
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

			service := handlersmocks.NewMockItemUpdateService(t)

			if testCase.serviceMock != nil {
				service.EXPECT().
					Exec(mock.Anything, &services.ItemUpdateRequest{
						ID:          uuid.MustParse(testCase.request.GetId()),
						Name:        testCase.request.GetName(),
						Description: testCase.request.GetDescription(),
					}).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewItemUpdate(service)

			res, err := handler.ItemUpdate(t.Context(), testCase.request)
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
