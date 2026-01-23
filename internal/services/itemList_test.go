package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-template/internal/dao"
	"github.com/a-novel/service-template/internal/services"
	servicesmocks "github.com/a-novel/service-template/internal/services/mocks"
)

func TestItemList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type repositoryMock struct {
		resp []*dao.Item
		err  error
	}

	testCases := []struct {
		name string

		request *services.ItemListRequest

		repositoryMock *repositoryMock

		expect    []*services.Item
		expectErr error
	}{
		{
			name: "Success",

			request: &services.ItemListRequest{Limit: 10, Offset: 0},

			repositoryMock: &repositoryMock{
				resp: []*dao.Item{
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
						Name:      "item three",
						CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						Name:      "item one",
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expect: []*services.Item{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Name:      "item three",
					CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:      "item one",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Success/Empty",

			request: &services.ItemListRequest{Limit: 10, Offset: 0},

			repositoryMock: &repositoryMock{resp: []*dao.Item{}},

			expect: []*services.Item{},
		},
		{
			name: "Success/LimitDefaulted",

			request: &services.ItemListRequest{Limit: 0, Offset: 0},

			repositoryMock: &repositoryMock{resp: []*dao.Item{}},

			expect: []*services.Item{},
		},
		{
			name: "Error/LimitTooHigh",

			request:   &services.ItemListRequest{Limit: 101, Offset: 0},
			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/OffsetNegative",

			request:   &services.ItemListRequest{Limit: 10, Offset: -1},
			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/Repository",

			request: &services.ItemListRequest{Limit: 10, Offset: 0},

			repositoryMock: &repositoryMock{err: errFoo},
			expectErr:      errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repository := servicesmocks.NewMockItemListRepository(t)

			if testCase.repositoryMock != nil {
				expectLimit := testCase.request.Limit
				if expectLimit <= 0 {
					expectLimit = services.ItemListDefaultSize
				}

				repository.EXPECT().
					Exec(mock.Anything, &dao.ItemListRequest{
						Limit:  expectLimit,
						Offset: testCase.request.Offset,
					}).
					Return(testCase.repositoryMock.resp, testCase.repositoryMock.err)
			}

			service := services.NewItemList(repository)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			repository.AssertExpectations(t)
		})
	}
}
