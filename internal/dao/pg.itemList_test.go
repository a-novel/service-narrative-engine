package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-template/internal/config/configtest"
	"github.com/a-novel/service-template/internal/dao"
	"github.com/a-novel/service-template/internal/models/migrations"
)

func TestPgItemList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		fixtures []*dao.Item

		request *dao.ItemListRequest

		expect    []*dao.Item
		expectErr error
	}{
		{
			name: "Success",

			fixtures: []*dao.Item{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:      "item one",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Name:      "item two",
					CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Name:      "item three",
					CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ItemListRequest{Limit: 10, After: uuid.Max},

			expect: []*dao.Item{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Name:      "item three",
					CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Name:      "item two",
					CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
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

			request: &dao.ItemListRequest{Limit: 10, After: uuid.Max},
		},
		{
			name: "Success/Limit",

			fixtures: []*dao.Item{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:      "item one",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Name:      "item two",
					CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Name:      "item three",
					CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ItemListRequest{Limit: 2, After: uuid.Max},

			expect: []*dao.Item{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Name:      "item three",
					CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Name:      "item two",
					CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Success/After",

			fixtures: []*dao.Item{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:      "item one",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Name:      "item two",
					CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Name:      "item three",
					CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ItemListRequest{Limit: 10, After: uuid.MustParse("00000000-0000-0000-0000-000000000003")},

			expect: []*dao.Item{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Name:      "item two",
					CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
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
			// The previous page ended on an item deleted since. Its ID still marks the position, so the
			// next page neither repeats nor skips a row.
			name: "Success/AfterDeletedItem",

			fixtures: []*dao.Item{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:      "item one",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Name:      "item three",
					CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ItemListRequest{Limit: 10, After: uuid.MustParse("00000000-0000-0000-0000-000000000002")},

			expect: []*dao.Item{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:      "item one",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	daoItemList := dao.NewPgItemList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				if len(testCase.fixtures) > 0 {
					_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
					require.NoError(t, err)
				}

				result, err := daoItemList.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
