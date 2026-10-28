package dao_test

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgCellUpsert(t *testing.T) {
	t.Parallel()

	content := json.RawMessage(`{"text": "Marie, Paul"}`)
	manuscriptContent := json.RawMessage(`{"blocks": ["Marie attend."]}`)

	testCases := []struct {
		name string

		request *dao.CellUpsertRequest

		// expect lists the written cells by scene, then layer.
		expect    []*dao.Cell
		expectErr error
	}{
		{
			name: "Success/Create",

			request: &dao.CellUpsertRequest{
				ProjectID: projectID,
				Cells:     []dao.CellUpsertItem{{SceneID: sceneID2, LayerID: notesLayerID, Content: content}},
				Now:       now,
			},

			expect: []*dao.Cell{
				{ProjectID: projectID, SceneID: sceneID2, LayerID: notesLayerID, Content: content, UpdatedAt: now},
			},
		},
		{
			name: "Success/Update",

			request: &dao.CellUpsertRequest{
				ProjectID: projectID,
				Cells:     []dao.CellUpsertItem{{SceneID: sceneID1, LayerID: notesLayerID, Content: content}},
				Now:       now,
			},

			expect: []*dao.Cell{
				{ProjectID: projectID, SceneID: sceneID1, LayerID: notesLayerID, Content: content, UpdatedAt: now},
			},
		},
		{
			name: "Success/Batch",

			request: &dao.CellUpsertRequest{
				ProjectID: projectID,
				Cells: []dao.CellUpsertItem{
					{SceneID: sceneID2, LayerID: notesLayerID, Content: content},
					{SceneID: sceneID1, LayerID: manuscriptLayerID, Content: manuscriptContent},
				},
				Now: now,
			},

			expect: []*dao.Cell{
				{
					ProjectID: projectID, SceneID: sceneID1, LayerID: manuscriptLayerID, Content: manuscriptContent,
					UpdatedAt: now,
				},
				{ProjectID: projectID, SceneID: sceneID2, LayerID: notesLayerID, Content: content, UpdatedAt: now},
			},
		},
		{
			name: "Error/UnknownScene",

			request: &dao.CellUpsertRequest{
				ProjectID: projectID,
				Cells:     []dao.CellUpsertItem{{SceneID: absentID, LayerID: notesLayerID, Content: content}},
				Now:       now,
			},

			expectErr: dao.ErrCellUpsertSceneNotFound,
		},
		{
			name: "Error/OtherProjectScene",

			request: &dao.CellUpsertRequest{
				ProjectID: projectID,
				Cells:     []dao.CellUpsertItem{{SceneID: otherSceneID, LayerID: notesLayerID, Content: content}},
				Now:       now,
			},

			expectErr: dao.ErrCellUpsertSceneNotFound,
		},
		{
			name: "Error/OtherProjectLayer",

			request: &dao.CellUpsertRequest{
				ProjectID: projectID,
				Cells:     []dao.CellUpsertItem{{SceneID: sceneID2, LayerID: otherLayerID, Content: content}},
				Now:       now,
			},

			expectErr: dao.ErrCellUpsertLayerNotFound,
		},
		{
			name: "Error/MixedBatch",

			request: &dao.CellUpsertRequest{
				ProjectID: projectID,
				Cells: []dao.CellUpsertItem{
					{SceneID: sceneID1, LayerID: notesLayerID, Content: content},
					{SceneID: absentID, LayerID: notesLayerID, Content: content},
				},
				Now: now,
			},

			expectErr: dao.ErrCellUpsertSceneNotFound,
		},
	}

	daoCellUpsert := dao.NewPgCellUpsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoCellUpsert.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				// A failed batch writes none of its cells, the valid ones included.
				if testCase.expectErr != nil {
					requireCellsEqual(t, composerFixtures().cells, listCells(ctx, t, projectID))

					return
				}

				slices.SortFunc(result, func(a, b *dao.CellUpsertRow) int {
					return cmp.Or(bytes.Compare(a.SceneID[:], b.SceneID[:]), bytes.Compare(a.LayerID[:], b.LayerID[:]))
				})
				require.Equal(t, lo.Map(testCase.expect, func(cell *dao.Cell, _ int) *dao.CellUpsertRow {
					return &dao.CellUpsertRow{SceneID: cell.SceneID, LayerID: cell.LayerID, UpdatedAt: cell.UpdatedAt}
				}), result)

				// The write returns the keys alone, so the stored cells carry the content.
				stored := lo.Filter(listCells(ctx, t, projectID), func(cell *dao.Cell, _ int) bool {
					return slices.ContainsFunc(testCase.expect, func(expect *dao.Cell) bool {
						return expect.SceneID == cell.SceneID && expect.LayerID == cell.LayerID
					})
				})
				requireCellsEqual(t, testCase.expect, stored)
			})
		})
	}
}
