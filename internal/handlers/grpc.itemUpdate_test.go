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

func TestGrpcItemUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		resp *core.Item
		err  error
	}

	testCases := []struct {
		name string

		request *templatev0.ItemUpdateRequest

		serviceMock *serviceMock

		expect       *templatev0.ItemUpdateResponse
		expectStatus codes.Code
	}{
		{
			name: "Success",

			request: &templatev0.ItemUpdateRequest{
				Id:          "00000000-0000-0000-0000-000000000001",
				Name:        "updated item",
				Description: "updated description",
			},

			serviceMock: &serviceMock{
				resp: &core.Item{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "updated item",
					Description: "updated description",
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			expectStatus: codes.OK,
			expect: &templatev0.ItemUpdateResponse{
				Item: &templatev0.Item{
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

			request: &templatev0.ItemUpdateRequest{
				Id:   "not-a-uuid",
				Name: "updated item",
			},

			expectStatus: codes.InvalidArgument,
		},
		{
			name: "Error/InvalidArgument",

			request: &templatev0.ItemUpdateRequest{
				Id:   "00000000-0000-0000-0000-000000000001",
				Name: "",
			},

			serviceMock: &serviceMock{
				err: core.ErrInvalidRequest,
			},

			expectStatus: codes.InvalidArgument,
		},
		{
			name: "Error/NotFound",

			request: &templatev0.ItemUpdateRequest{
				Id:   "00000000-0000-0000-0000-000000000001",
				Name: "updated item",
			},

			serviceMock: &serviceMock{
				err: core.ErrItemUpdateNotFound,
			},

			expectStatus: codes.NotFound,
		},
		{
			name: "Error/Internal",

			request: &templatev0.ItemUpdateRequest{
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

			service := handlersmocks.NewMockGrpcItemUpdateService(t)

			if testCase.serviceMock != nil {
				serviceRequest := &core.ItemUpdateRequest{
					ID:          uuid.MustParse(testCase.request.GetId()),
					Name:        testCase.request.GetName(),
					Description: testCase.request.GetDescription(),
				}

				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewGrpcItemUpdate(service)

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
