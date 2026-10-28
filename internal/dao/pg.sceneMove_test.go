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

func TestPgSceneMove(t *testing.T) {
	t.Parallel()

	scenes := composerFixtures().scenes

	testCases := []struct {
		name string

		request *dao.SceneMoveRequest

		expect       *dao.Scene
		expectScenes []*dao.Scene
		expectErr    error
	}{
		{
			name: "Success/Up",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: sceneID3, Position: 0},

			expect:       sceneAt(scenes[2], 0),
			expectScenes: []*dao.Scene{sceneAt(scenes[2], 0), sceneAt(scenes[0], 1), sceneAt(scenes[1], 2)},
		},
		{
			name: "Success/Down",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: sceneID1, Position: 1},

			expect:       sceneAt(scenes[0], 1),
			expectScenes: []*dao.Scene{sceneAt(scenes[1], 0), sceneAt(scenes[0], 1), scenes[2]},
		},
		{
			name: "Success/PastTheEnd",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: sceneID1, Position: 99},

			expect:       sceneAt(scenes[0], 2),
			expectScenes: []*dao.Scene{sceneAt(scenes[1], 0), sceneAt(scenes[2], 1), sceneAt(scenes[0], 2)},
		},
		{
			name: "Success/SamePosition",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: sceneID2, Position: 1},

			expect:       scenes[1],
			expectScenes: scenes[:3],
		},
		{
			name: "Error/NotFound",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: absentID, Position: 0},

			expectErr: dao.ErrSceneMoveNotFound,
		},
		{
			name: "Error/OtherProject",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: otherSceneID, Position: 0},

			expectErr: dao.ErrSceneMoveNotFound,
		},
	}

	daoSceneMove := dao.NewPgSceneMove()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoSceneMove.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				if testCase.expectErr == nil {
					require.Equal(t, testCase.expectScenes, listScenes(ctx, t, projectID))
				}
			})
		})
	}
}
