package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestItemList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		resp []*dao.Item
		err  error
	}

	testCases := []struct {
		name string

		request *core.ItemListRequest

		daoMock *daoMock

		expect    []*core.Item
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ItemListRequest{Limit: 10, Offset: 0},

			daoMock: &daoMock{
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

			expect: []*core.Item{
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

			request: &core.ItemListRequest{Limit: 10, Offset: 0},

			daoMock: &daoMock{resp: []*dao.Item{}},

			expect: []*core.Item{},
		},
		{
			name: "Success/LimitDefaulted",

			request: &core.ItemListRequest{Limit: 0, Offset: 0},

			daoMock: &daoMock{resp: []*dao.Item{}},

			expect: []*core.Item{},
		},
		{
			name: "Error/LimitTooHigh",

			request:   &core.ItemListRequest{Limit: 101, Offset: 0},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/OffsetNegative",

			request:   &core.ItemListRequest{Limit: 10, Offset: -1},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Dao",

			request: &core.ItemListRequest{Limit: 10, Offset: 0},

			daoMock:   &daoMock{err: errFoo},
			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockItemListDao(t)

			if testCase.daoMock != nil {
				expectLimit := testCase.request.Limit
				if expectLimit <= 0 {
					expectLimit = core.ItemListDefaultSize
				}

				mockDao.EXPECT().
					Exec(mock.Anything, &dao.ItemListRequest{
						Limit:  expectLimit,
						Offset: testCase.request.Offset,
					}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewItemList(mockDao)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			mockDao.AssertExpectations(t)
		})
	}
}
