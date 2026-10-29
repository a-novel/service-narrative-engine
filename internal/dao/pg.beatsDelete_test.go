package dao_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgBeatsDelete(t *testing.T) {
	t.Parallel()

	// A fourth beat, with a scene and a cell, lets a delete leave a gap before a beat that has to move up
	// twice.
	beat4 := &dao.Beat{
		ID: uuid.MustParse("00000000-0000-0000-0000-000000000034"), ProjectID: projectID, Position: 3,
		Title: "Le débat", Color: beatColor, UpdatedAt: updatedAt,
	}
	beat4Scene := &dao.Scene{
		ID: uuid.MustParse("00000000-0000-0000-0000-000000000017"), ProjectID: projectID, BeatID: beat4.ID,
		Position: 0, Title: "Le débat", UpdatedAt: updatedAt,
	}
	beat4Cell := &dao.Cell{
		ProjectID: projectID, SceneID: beat4Scene.ID, LayerID: notesLayerID, Content: json.RawMessage(`{"text": "Paul"}`),
		UpdatedAt: updatedAt,
	}

	// beats and cells are the test project's rows before the delete, in order.
	fixtureBeats, fixtureCells := composerFixtures().beats, composerFixtures().cells
	beats := []*dao.Beat{fixtureBeats[0], fixtureBeats[1], fixtureBeats[2], beat4}
	cells := []*dao.Cell{fixtureCells[0], fixtureCells[1], beat4Cell}

	testCases := []struct {
		name string

		request *dao.BeatsDeleteRequest

		expect      []*dao.Beat
		expectBeats []*dao.Beat
		expectCells []*dao.Cell
	}{
		{
			name: "Success",

			request: &dao.BeatsDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{beatID1}},

			expect:      []*dao.Beat{beats[0]},
			expectBeats: []*dao.Beat{beatAt(beats[1], 0), beatAt(beats[2], 1), beatAt(beat4, 2)},
			expectCells: []*dao.Cell{beat4Cell},
		},
		{
			name: "Success/Middle",

			request: &dao.BeatsDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{beatID2}},

			expect:      []*dao.Beat{beats[1]},
			expectBeats: []*dao.Beat{beats[0], beatAt(beats[2], 1), beatAt(beat4, 2)},
			expectCells: cells,
		},
		{
			name: "Success/NonAdjacent",

			request: &dao.BeatsDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{beatID1, beatID3}},

			expect:      []*dao.Beat{beats[0], beats[2]},
			expectBeats: []*dao.Beat{beatAt(beats[1], 0), beatAt(beat4, 1)},
			expectCells: []*dao.Cell{beat4Cell},
		},
		{
			name: "Success/NotFound",

			request: &dao.BeatsDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{absentID}},

			expectBeats: beats,
			expectCells: cells,
		},
		{
			name: "Success/OtherProject",

			request: &dao.BeatsDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{otherBeatID}},

			expectBeats: beats,
			expectCells: cells,
		},
	}

	daoBeatsDelete := dao.NewPgBeatsDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				data.beats = append(data.beats, beat4)
				data.scenes = append(data.scenes, beat4Scene)
				data.cells = append(data.cells, beat4Cell)

				insertFixtures(ctx, t, data)

				result, err := daoBeatsDelete.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.ElementsMatch(t, testCase.expect, result)

				// The beats after the deleted ones move up, and the deleted ones' scenes and cells go with them.
				require.Equal(t, testCase.expectBeats, listBeats(ctx, t, projectID))
				requireCellsEqual(t, testCase.expectCells, listCells(ctx, t, projectID))
			})
		})
	}
}
