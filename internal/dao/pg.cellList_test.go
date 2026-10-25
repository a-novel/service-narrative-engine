package dao_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgCellList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.CellListRequest

		expect    []*dao.Cell
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.CellListRequest{ProjectID: projectID, Limit: 10},

			expect: composerFixtures().cells,
		},
		{
			name: "Success/Limit",

			request: &dao.CellListRequest{ProjectID: projectID, Limit: 1},

			expect: composerFixtures().cells[:1],
		},
		{
			name: "Success/OtherProject",

			request: &dao.CellListRequest{ProjectID: otherProjectID, Limit: 10},
		},
	}

	daoCellList := dao.NewPgCellList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoCellList.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				requireCellsEqual(t, testCase.expect, result)
			})
		})
	}
}
