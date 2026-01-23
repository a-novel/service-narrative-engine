package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-template/internal/config/configtest"
	"github.com/a-novel/service-template/internal/dao"
)

func TestItemCreate(t *testing.T) {
	testCases := []struct {
		name string

		request *dao.ItemCreateRequest

		expect    *dao.Item
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ItemCreateRequest{
				ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:        "test item",
				Description: "test description",
				Now:         time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.Item{
				ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:        "test item",
				Description: "test description",
				CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "NoDescription",

			request: &dao.ItemCreateRequest{
				ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name: "test item no desc",
				Now:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.Item{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:      "test item no desc",
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	repository := dao.NewItemCreate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			postgres.RunTransactionalTest(t, configtest.PostgresPreset, func(ctx context.Context, t *testing.T) {
				t.Helper()

				result, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
