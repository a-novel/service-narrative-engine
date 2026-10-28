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

	layers := composerFixtures().layers

	testCases := []struct {
		name string

		request *dao.LayerDeleteRequest

		expect       *dao.Layer
		expectLayers []*dao.Layer
		expectCells  []*dao.Cell
		expectErr    error
	}{
		{
			name: "Success",

			request: &dao.LayerDeleteRequest{ID: notesLayerID},

			expect:       layers[1],
			expectLayers: []*dao.Layer{layers[0]},
			expectCells:  composerFixtures().cells[:1],
		},
		{
			name: "Success/First",

			request: &dao.LayerDeleteRequest{ID: manuscriptLayerID},

			expect:       layers[0],
			expectLayers: []*dao.Layer{layerAt(layers[1], 0)},
			expectCells:  composerFixtures().cells[1:],
		},
		{
			name: "Error/NotFound",

			request: &dao.LayerDeleteRequest{ID: absentID},

			expectLayers: layers[:2],
			expectCells:  composerFixtures().cells,
			expectErr:    dao.ErrLayerDeleteNotFound,
		},
	}

	daoLayerDelete := dao.NewPgLayerDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoLayerDelete.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				// The layers after the deleted one move up, and its cells go with it.
				require.Equal(t, testCase.expectLayers, listLayers(ctx, t, projectID))
				requireCellsEqual(t, testCase.expectCells, listCells(ctx, t, projectID))
			})
		})
	}
}
