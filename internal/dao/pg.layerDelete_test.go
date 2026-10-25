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

func TestPgLayerDelete(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.LayerDeleteRequest

		expect      *dao.Layer
		expectCells []*dao.Cell
		expectErr   error
	}{
		{
			name: "Success",

			request: &dao.LayerDeleteRequest{ID: notesLayerID, OwnerID: ownerID, Revision: 1},

			expect:      composerFixtures().layers[1],
			expectCells: composerFixtures().cells[:1],
		},
		{
			name: "Error/StaleRevision",

			request: &dao.LayerDeleteRequest{ID: notesLayerID, OwnerID: ownerID, Revision: 2},

			expectCells: composerFixtures().cells,
			expectErr:   dao.ErrLayerDeleteNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: &dao.LayerDeleteRequest{ID: notesLayerID, OwnerID: otherOwnerID, Revision: 1},

			expectCells: composerFixtures().cells,
			expectErr:   dao.ErrLayerDeleteNotFound,
		},
	}

	daoLayerDelete := dao.NewPgLayerDelete()
	daoCellList := dao.NewPgCellList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoLayerDelete.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				// Deleting a layer deletes its cells with it.
				cells, err := daoCellList.Exec(ctx, &dao.CellListRequest{ProjectID: projectID, Limit: 10})
				require.NoError(t, err)
				requireCellsEqual(t, testCase.expectCells, cells)
			})
		})
	}
}
