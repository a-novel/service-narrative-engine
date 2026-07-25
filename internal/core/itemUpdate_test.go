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

func TestItemUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		resp *dao.Item
		err  error
	}

	testCases := []struct {
		name string

		request *core.ItemUpdateRequest

		daoMock *daoMock

		expect    *core.Item
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ItemUpdateRequest{
				Actor:       testActor,
				ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:        "updated item",
				Description: "updated description",
			},

			daoMock: &daoMock{
				resp: &dao.Item{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "updated item",
					Description: "updated description",
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &core.Item{
				ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:        "updated item",
				Description: "updated description",
				CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error/MissingActor",

			request: &core.ItemUpdateRequest{
				ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name: "updated item",
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/EmptyName",

			request: &core.ItemUpdateRequest{
				Actor: testActor,
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:  "",
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/WhitespaceOnlyName",

			request: &core.ItemUpdateRequest{
				Actor: testActor,
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:  "   ",
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NameTooLong",

			request: &core.ItemUpdateRequest{
				Actor: testActor,
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:  string(make([]byte, 257)),
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Dao",

			request: &core.ItemUpdateRequest{
				Actor: testActor,
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:  "updated item",
			},

			daoMock:   &daoMock{err: errFoo},
			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockItemUpdateDao(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ItemUpdateRequest) bool {
						return assert.WithinDuration(t, time.Now(), req.Now, time.Minute) &&
							assert.Equal(t, testCase.request.ID, req.ID) &&
							assert.Equal(t, testCase.request.Name, req.Name) &&
							assert.Equal(t, testCase.request.Description, req.Description)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewItemUpdate(mockDao)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			mockDao.AssertExpectations(t)
		})
	}
}
