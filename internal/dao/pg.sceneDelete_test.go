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

func TestPgSceneDelete(t *testing.T) {
	t.Parallel()

	scenes := composerFixtures().scenes

	testCases := []struct {
		name string

		request *dao.SceneDeleteRequest

		expect       *dao.Scene
		expectScenes []*dao.Scene
		expectCells  []*dao.Cell
		expectErr    error
	}{
		{
			name: "Success",

			request: &dao.SceneDeleteRequest{ProjectID: projectID, ID: sceneID1},

			expect:       scenes[0],
			expectScenes: []*dao.Scene{sceneAt(scenes[1], 0), sceneAt(scenes[2], 1), scenes[3]},
			expectCells:  []*dao.Cell{},
		},
		{
			name: "Success/Middle",

			request: &dao.SceneDeleteRequest{ProjectID: projectID, ID: sceneID2},

			expect:       scenes[1],
			expectScenes: []*dao.Scene{scenes[0], sceneAt(scenes[2], 1), scenes[3]},
			expectCells:  composerFixtures().cells,
		},
		{
			name: "Error/NotFound",

			request: &dao.SceneDeleteRequest{ProjectID: projectID, ID: absentID},

			expectScenes: scenes[:4],
			expectCells:  composerFixtures().cells,
			expectErr:    dao.ErrSceneDeleteNotFound,
		},
		{
			name: "Error/OtherProject",

			request: &dao.SceneDeleteRequest{ProjectID: projectID, ID: otherSceneID},

			expectScenes: scenes[:4],
			expectCells:  composerFixtures().cells,
			expectErr:    dao.ErrSceneDeleteNotFound,
		},
	}

	daoSceneDelete := dao.NewPgSceneDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoSceneDelete.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				// The scenes after the deleted one in its beat move up, and its cells go with it.
				require.Equal(t, testCase.expectScenes, listScenes(ctx, t, projectID))
				requireCellsEqual(t, testCase.expectCells, listCells(ctx, t, projectID))
			})
		})
	}
}
