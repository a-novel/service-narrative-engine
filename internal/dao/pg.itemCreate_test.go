package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestItemCreate(t *testing.T) {
	t.Parallel()

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

	dao := dao.NewItemCreate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				result, err := dao.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
