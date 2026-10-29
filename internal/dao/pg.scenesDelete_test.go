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

func TestPgScenesDelete(t *testing.T) {
	t.Parallel()

	// A fourth scene in the first beat, with a cell, lets a delete leave a gap before a scene that has to move
	// up twice.
	extra := &dao.Scene{
		ID: uuid.MustParse("00000000-0000-0000-0000-000000000014"), ProjectID: projectID, BeatID: beatID1,
		Position: 3, Title: "Le débat", UpdatedAt: updatedAt,
	}
	extraCell := &dao.Cell{
		ProjectID: projectID, SceneID: extra.ID, LayerID: notesLayerID, Content: json.RawMessage(`{"text": "Paul"}`),
		UpdatedAt: updatedAt,
	}

	// scenes and cells are the test project's rows before the delete, in narrative order.
	fixtureScenes, fixtureCells := composerFixtures().scenes, composerFixtures().cells
	scenes := []*dao.Scene{fixtureScenes[0], fixtureScenes[1], fixtureScenes[2], extra, fixtureScenes[3]}
	cells := []*dao.Cell{fixtureCells[0], fixtureCells[1], extraCell}

	testCases := []struct {
		name string

		request *dao.ScenesDeleteRequest

		expect       []*dao.Scene
		expectScenes []*dao.Scene
		expectCells  []*dao.Cell
	}{
		{
			name: "Success",

			request: &dao.ScenesDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{sceneID1}},

			expect:       []*dao.Scene{scenes[0]},
			expectScenes: []*dao.Scene{sceneAt(scenes[1], 0), sceneAt(scenes[2], 1), sceneAt(extra, 2), scenes[4]},
			expectCells:  []*dao.Cell{extraCell},
		},
		{
			name: "Success/Middle",

			request: &dao.ScenesDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{sceneID2}},

			expect:       []*dao.Scene{scenes[1]},
			expectScenes: []*dao.Scene{scenes[0], sceneAt(scenes[2], 1), sceneAt(extra, 2), scenes[4]},
			expectCells:  cells,
		},
		{
			name: "Success/NonAdjacent",

			request: &dao.ScenesDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{sceneID1, sceneID3}},

			expect:       []*dao.Scene{scenes[0], scenes[2]},
			expectScenes: []*dao.Scene{sceneAt(scenes[1], 0), sceneAt(extra, 1), scenes[4]},
			expectCells:  []*dao.Cell{extraCell},
		},
		{
			name: "Success/AcrossBeats",

			request: &dao.ScenesDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{sceneID2, sceneID4}},

			expect:       []*dao.Scene{scenes[1], scenes[4]},
			expectScenes: []*dao.Scene{scenes[0], sceneAt(scenes[2], 1), sceneAt(extra, 2)},
			expectCells:  cells,
		},
		{
			name: "Success/NotFound",

			request: &dao.ScenesDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{absentID}},

			expectScenes: scenes,
			expectCells:  cells,
		},
		{
			name: "Success/OtherProject",

			request: &dao.ScenesDeleteRequest{ProjectID: projectID, IDs: []uuid.UUID{otherSceneID}},

			expectScenes: scenes,
			expectCells:  cells,
		},
	}

	daoScenesDelete := dao.NewPgScenesDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				data.scenes = append(data.scenes, extra)
				data.cells = append(data.cells, extraCell)

				insertFixtures(ctx, t, data)

				result, err := daoScenesDelete.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.ElementsMatch(t, testCase.expect, result)

				// The scenes after the deleted ones in their beats move up, and their cells go with them.
				require.Equal(t, testCase.expectScenes, listScenes(ctx, t, projectID))
				requireCellsEqual(t, testCase.expectCells, listCells(ctx, t, projectID))
			})
		})
	}
}
