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

	testCases := []struct {
		name string

		request *dao.CellSummaryListRequest

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

				result, err := daoCellSummaryList.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
