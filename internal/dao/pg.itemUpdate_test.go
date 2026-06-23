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

func TestItemUpdate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		fixtures []*dao.Item

		request *dao.ItemUpdateRequest

		expect    *dao.Item
		expectErr error
	}{
		{
			name: "Success",

			fixtures: []*dao.Item{
				{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "original item",
					Description: "original description",
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ItemUpdateRequest{
				ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:        "updated item",
				Description: "updated description",
				Now:         time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.Item{
				ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:        "updated item",
				Description: "updated description",
				CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error/NotFound",

			request: &dao.ItemUpdateRequest{
				ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name: "updated item",
				Now:  time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},

			expectErr: dao.ErrItemUpdateNotFound,
		},
	}

	repository := dao.NewItemUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				if len(testCase.fixtures) > 0 {
					_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
					require.NoError(t, err)
				}

				result, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
