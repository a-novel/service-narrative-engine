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

func TestGrpcItemCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceMock struct {
		resp *core.Item
		err  error
	}

	testCases := []struct {
		name string

		request *templatev0.ItemCreateRequest

		serviceMock *serviceMock

		expect       *templatev0.ItemCreateResponse
		expectStatus codes.Code
	}{
		{
			name: "Success",

			request: &templatev0.ItemCreateRequest{
				Name:        "test item",
				Description: "test description",
			},

			serviceMock: &serviceMock{
				resp: &core.Item{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "test item",
					Description: "test description",
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expectStatus: codes.OK,
			expect: &templatev0.ItemCreateResponse{
				Item: &templatev0.Item{
					Id:          "00000000-0000-0000-0000-000000000001",
					Name:        "test item",
					Description: "test description",
					CreatedAt:   "2021-01-01T00:00:00Z",
					UpdatedAt:   "2021-01-01T00:00:00Z",
				},
			},
		},
		{
			name: "Error/InvalidArgument",

			request: &templatev0.ItemCreateRequest{
				Name: "",
			},

			serviceMock: &serviceMock{
				err: core.ErrInvalidRequest,
			},

			expectStatus: codes.InvalidArgument,
		},
		{
			name: "Error/Internal",

			request: &templatev0.ItemCreateRequest{
				Name: "test item",
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

			service := handlersmocks.NewMockGrpcItemCreateService(t)

			if testCase.serviceMock != nil {
				serviceRequest := &core.ItemCreateRequest{
					Name:        testCase.request.GetName(),
					Description: testCase.request.GetDescription(),
				}

				service.EXPECT().
					Exec(mock.Anything, serviceRequest).
					Return(testCase.serviceMock.resp, testCase.serviceMock.err)
			}

			handler := handlers.NewGrpcItemCreate(service)

			res, err := handler.ItemCreate(t.Context(), testCase.request)
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
