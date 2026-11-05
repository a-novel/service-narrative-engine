package dao_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

// summaryCells returns cells of every shape a summary measures: a manuscript with a generated mark, a
// repeated speaker, a tone and accented text, and a recap; a notes cell with characters to escape; an empty
// cell of each kind; and a cell of another project.
func summaryCells() []*dao.Cell {
	return []*dao.Cell{
		{
			ProjectID: projectID, SceneID: sceneID1, LayerID: manuscriptLayerID, UpdatedAt: updatedAt,
			Content: json.RawMessage(`{"blocks": [
				{"kind": "narration", "text": "Élodie rit.", "generated": {"model": "gpt-6.1-terra"}},
				{"kind": "dialogue", "speaker": "Marie", "parts": [{"tone": "bas"}, {"text": "Déjà ?"}]},
				{"kind": "dialogue", "speaker": "Paul", "parts": [{"text": "Oui."}]},
				{"kind": "dialogue", "speaker": "Marie", "parts": []}
			]}`),
			Recap: &dao.Recap{
				Condensed: "Marie rit.", Compact: "Rires.", Essentials: "Rire.", Names: []string{"Marie", "Paul"},
			},
			RecapCellUpdatedAt: &createdAt,
		},
		{
			ProjectID: projectID, SceneID: sceneID1, LayerID: notesLayerID, UpdatedAt: updatedAt,
			Content: json.RawMessage(`{"text": "Marie & \"Paul\""}`),
		},
		{
			ProjectID: projectID, SceneID: sceneID2, LayerID: manuscriptLayerID, UpdatedAt: updatedAt,
			Content: json.RawMessage(`{"blocks": []}`),
		},
		{
			ProjectID: projectID, SceneID: sceneID2, LayerID: notesLayerID, UpdatedAt: updatedAt,
			Content: json.RawMessage(`{"text": ""}`),
		},
		{
			ProjectID: otherProjectID, SceneID: otherSceneID, LayerID: otherLayerID, UpdatedAt: updatedAt,
			Content: json.RawMessage(`{"blocks": [{"kind": "narration", "text": "Other."}]}`),
		},
	}
}

func TestPgCellSummaryList(t *testing.T) {
	t.Parallel()

	// The summaries of the test project, by layer, then by scene. The size of the manuscript counts
	// "Élodie rit.", "Marie", "bas", "Déjà ?", "Paul", "Oui." and "Marie" again, but no kind and no model.
	summaries := []*dao.CellSummaryListRow{
		{
			SceneID: sceneID1, LayerID: manuscriptLayerID, UpdatedAt: updatedAt, Size: 38,
			Speakers:   []string{"Marie", "Paul"},
			RecapNames: []string{"Marie", "Paul"}, RecapCondensedSize: lo.ToPtr(10), RecapCompactSize: lo.ToPtr(6),
			RecapEssentialsSize: lo.ToPtr(5), RecapCellUpdatedAt: &createdAt,
		},
		{SceneID: sceneID2, LayerID: manuscriptLayerID, UpdatedAt: updatedAt, Empty: true, Speakers: []string{}},
		{SceneID: sceneID1, LayerID: notesLayerID, UpdatedAt: updatedAt, Size: 14, Speakers: []string{}},
		{SceneID: sceneID2, LayerID: notesLayerID, UpdatedAt: updatedAt, Empty: true, Speakers: []string{}},
	}

	for _, summary := range summaries {
		summary.Total = len(summaries)
	}

	// The summaries once the manuscript of the second scene and the recap of the first are rewritten.
	rewritten := []*dao.CellSummaryListRow{
		{
			SceneID: sceneID1, LayerID: manuscriptLayerID, UpdatedAt: updatedAt, Size: 38,
			Speakers:   []string{"Marie", "Paul"},
			RecapNames: []string{"Paul"}, RecapCondensedSize: lo.ToPtr(10), RecapCompactSize: lo.ToPtr(5),
			RecapEssentialsSize: lo.ToPtr(3), RecapCellUpdatedAt: &updatedAt, Total: 4,
		},
		{SceneID: sceneID2, LayerID: manuscriptLayerID, UpdatedAt: now, Size: 6, Speakers: []string{"Léa"}, Total: 4},
		summaries[2],
		summaries[3],
	}

	testCases := []struct {
		name string

		request *dao.CellSummaryListRequest
		// rewrite writes cells through the write DAOs before the page is read.
		rewrite func(ctx context.Context, t *testing.T)

		expect []*dao.CellSummaryListRow
	}{
		{
			name: "Success",

			request: &dao.CellSummaryListRequest{ProjectID: projectID, Limit: 10},

			expect: summaries,
		},
		{
			name: "Success/Page",

			request: &dao.CellSummaryListRequest{ProjectID: projectID, Limit: 2, Offset: 1},

			expect: summaries[1:3],
		},
		{
			name: "Success/PastEnd",

			request: &dao.CellSummaryListRequest{ProjectID: projectID, Limit: 10, Offset: 4},
		},
		{
			name: "Success/OtherProject",

			request: &dao.CellSummaryListRequest{ProjectID: absentID, Limit: 10},
		},
		{
			// A write of the content or of the recap refreshes the summary in the statement that writes.
			name: "Success/Rewritten",

			request: &dao.CellSummaryListRequest{ProjectID: projectID, Limit: 10},
			rewrite: func(ctx context.Context, t *testing.T) {
				t.Helper()

				_, err := dao.NewPgCellUpsert().Exec(ctx, &dao.CellUpsertRequest{
					ProjectID: projectID, SceneID: sceneID2, LayerID: manuscriptLayerID, Now: now,
					Content: json.RawMessage(`{"blocks": [{"kind": "dialogue", "speaker": "Léa", "parts": [{"text": "Là."}]}]}`),
				})
				require.NoError(t, err)

				err = dao.NewPgCellRecapUpdate().Exec(ctx, &dao.CellRecapUpdateRequest{
					ProjectID: projectID, SceneID: sceneID1, LayerID: manuscriptLayerID, CellUpdatedAt: updatedAt,
					Recap: json.RawMessage(`{"condensed": "Paul part.", "compact": "Part.", "essentials": "Va.", "names": ["Paul"]}`),
				})
				require.NoError(t, err)
			},

			expect: rewritten,
		},
	}

	daoCellSummaryList := dao.NewPgCellSummaryList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				data.cells = summaryCells()

				insertFixtures(ctx, t, data)

				if testCase.rewrite != nil {
					testCase.rewrite(ctx, t)
				}

				result, err := daoCellSummaryList.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
