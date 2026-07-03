package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestItemCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		resp *dao.Item
		err  error
	}

	testCases := []struct {
		name string

		request *core.ItemCreateRequest

		daoMock *daoMock

		expect    *core.Item
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ItemCreateRequest{
				Name:        "test item",
				Description: "test description",
			},

			daoMock: &daoMock{
				resp: &dao.Item{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "test item",
					Description: "test description",
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &core.Item{
				ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:        "test item",
				Description: "test description",
				CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error/EmptyName",

			request:   &core.ItemCreateRequest{Name: ""},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/WhitespaceOnlyName",

			request:   &core.ItemCreateRequest{Name: "   "},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NameTooLong",

			request:   &core.ItemCreateRequest{Name: string(make([]byte, 257))},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Dao",

			request: &core.ItemCreateRequest{Name: "test item"},

			daoMock:   &daoMock{err: errFoo},
			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockItemCreateDao(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ItemCreateRequest) bool {
						return assert.NotEqual(t, uuid.Nil, req.ID) &&
							assert.WithinDuration(t, time.Now(), req.Now, time.Minute) &&
							assert.Equal(t, testCase.request.Name, req.Name) &&
							assert.Equal(t, testCase.request.Description, req.Description)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewItemCreate(mockDao)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			mockDao.AssertExpectations(t)
		})
	}
}
