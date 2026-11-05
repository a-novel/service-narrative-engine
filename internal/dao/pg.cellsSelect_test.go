package dao_test

import (
	"cmp"
	"context"
	"slices"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgCellsSelect(t *testing.T) {
	t.Parallel()

	cells := summaryCells()
	recapped, notes, empty := cells[0], cells[1], cells[2]

	testCases := []struct {
		name string

		request *dao.CellsSelectRequest

		expect []*dao.CellsSelectRow
	}{
		{
			name: "Success",

			request: &dao.CellsSelectRequest{ProjectID: projectID, Cells: []*dao.CellsSelectKey{
				{SceneID: sceneID1, LayerID: manuscriptLayerID, Content: true, Recap: "compact"},
				{SceneID: sceneID1, LayerID: notesLayerID, Content: true},
			}},

			expect: []*dao.CellsSelectRow{
				{
					SceneID: sceneID1, LayerID: manuscriptLayerID, Content: recapped.Content, RecapVersion: "compact",
					Recap: lo.ToPtr("Rires."),
				},
				{SceneID: sceneID1, LayerID: notesLayerID, Content: notes.Content},
			},
		},
		{
			// A key that asks for one version leaves the content and the other versions unread, and a cell
			// without a recap has none.
			name: "Success/Recaps",

			request: &dao.CellsSelectRequest{ProjectID: projectID, Cells: []*dao.CellsSelectKey{
				{SceneID: sceneID1, LayerID: manuscriptLayerID, Recap: "essentials"},
				{SceneID: sceneID2, LayerID: manuscriptLayerID, Recap: "condensed"},
			}},

			expect: []*dao.CellsSelectRow{
				{SceneID: sceneID1, LayerID: manuscriptLayerID, RecapVersion: "essentials", Recap: lo.ToPtr("Rire.")},
				{SceneID: sceneID2, LayerID: manuscriptLayerID, RecapVersion: "condensed"},
			},
		},
		{
			name: "Success/Contents",

			request: &dao.CellsSelectRequest{ProjectID: projectID, Cells: []*dao.CellsSelectKey{
				{SceneID: sceneID1, LayerID: manuscriptLayerID, Content: true},
				{SceneID: sceneID2, LayerID: manuscriptLayerID, Content: true},
			}},

			expect: []*dao.CellsSelectRow{
				{SceneID: sceneID1, LayerID: manuscriptLayerID, Content: recapped.Content},
				{SceneID: sceneID2, LayerID: manuscriptLayerID, Content: empty.Content},
			},
		},
		{
			// A cell never written, and a cell of another project, read as no row.
			name: "Success/NotFound",

			request: &dao.CellsSelectRequest{ProjectID: projectID, Cells: []*dao.CellsSelectKey{
				{SceneID: sceneID3, LayerID: manuscriptLayerID, Content: true},
				{SceneID: otherSceneID, LayerID: otherLayerID, Content: true},
			}},
		},
	}

	daoCellsSelect := dao.NewPgCellsSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				data.cells = summaryCells()

				insertFixtures(ctx, t, data)

				result, err := daoCellsSelect.Exec(ctx, testCase.request)
				require.NoError(t, err)

				// The rows come in no particular order.
				slices.SortFunc(result, func(a, b *dao.CellsSelectRow) int {
					return cmp.Or(cmp.Compare(a.SceneID.String(), b.SceneID.String()),
						cmp.Compare(a.LayerID.String(), b.LayerID.String()))
				})

				require.Len(t, result, len(testCase.expect))

				for index, expect := range testCase.expect {
					require.Equal(t, expect.SceneID, result[index].SceneID)
					require.Equal(t, expect.LayerID, result[index].LayerID)
					require.Equal(t, expect.RecapVersion, result[index].RecapVersion)
					require.Equal(t, expect.Recap, result[index].Recap)

					if expect.Content == nil {
						require.Nil(t, result[index].Content)
					} else {
						require.JSONEq(t, string(expect.Content), string(result[index].Content))
					}
				}
			})
		})
	}
}
