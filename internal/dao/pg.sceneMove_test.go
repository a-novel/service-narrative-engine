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

	// The first beat holds the first three scenes, the second beat the fourth, and the third beat none.
	scenes := composerFixtures().scenes

	testCases := []struct {
		name string

		request *dao.SceneMoveRequest

		expect *dao.Scene
		// expectScenes is the project's scenes after the move, in narrative order.
		expectScenes []*dao.Scene
		// full fills the first beat up to its last position before the move.
		full      bool
		expectErr error
	}{
		{
			name: "Success/Up",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: sceneID3, BeatID: beatID1, Position: 0},

			expect: sceneAt(scenes[2], 0),
			expectScenes: []*dao.Scene{
				sceneAt(scenes[2], 0), sceneAt(scenes[0], 1), sceneAt(scenes[1], 2), scenes[3],
			},
		},
		{
			name: "Success/Down",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: sceneID1, BeatID: beatID1, Position: 1},

			expect:       sceneAt(scenes[0], 1),
			expectScenes: []*dao.Scene{sceneAt(scenes[1], 0), sceneAt(scenes[0], 1), scenes[2], scenes[3]},
		},
		{
			name: "Success/PastTheEnd",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: sceneID1, BeatID: beatID1, Position: 9},

			expect: sceneAt(scenes[0], 2),
			expectScenes: []*dao.Scene{
				sceneAt(scenes[1], 0), sceneAt(scenes[2], 1), sceneAt(scenes[0], 2), scenes[3],
			},
		},
		{
			name: "Success/SamePosition",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: sceneID2, BeatID: beatID1, Position: 1},

			expect:       scenes[1],
			expectScenes: scenes[:4],
		},
		{
			name: "Success/OtherBeat",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: sceneID2, BeatID: beatID2, Position: 0},

			expect: sceneIn(scenes[1], beatID2, 0),
			expectScenes: []*dao.Scene{
				scenes[0], sceneAt(scenes[2], 1), sceneIn(scenes[1], beatID2, 0), sceneAt(scenes[3], 1),
			},
		},
		{
			name: "Success/OtherBeatPastTheEnd",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: sceneID1, BeatID: beatID2, Position: 9},

			expect: sceneIn(scenes[0], beatID2, 1),
			expectScenes: []*dao.Scene{
				sceneAt(scenes[1], 0), sceneAt(scenes[2], 1), scenes[3], sceneIn(scenes[0], beatID2, 1),
			},
		},
		{
			name: "Success/EmptyBeat",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: sceneID4, BeatID: beatID3, Position: 0},

			expect:       sceneIn(scenes[3], beatID3, 0),
			expectScenes: []*dao.Scene{scenes[0], scenes[1], scenes[2], sceneIn(scenes[3], beatID3, 0)},
		},
		{
			name: "Error/NotFound",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: absentID, BeatID: beatID1, Position: 0},

			expectErr: dao.ErrSceneMoveNotFound,
		},
		{
			name: "Error/OtherProject",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: otherSceneID, BeatID: beatID1, Position: 0},

			expectErr: dao.ErrSceneMoveNotFound,
		},
		{
			name: "Error/BeatOfOtherProject",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: sceneID1, BeatID: otherBeatID, Position: 0},

			expectErr: dao.ErrSceneMoveBeatNotFound,
		},
		{
			name: "Error/BeatFull",

			request: &dao.SceneMoveRequest{ProjectID: projectID, ID: sceneID4, BeatID: beatID1, Position: 0},

			full:      true,
			expectErr: dao.ErrSceneMoveFull,
		},
	}

	daoSceneMove := dao.NewPgSceneMove()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				if testCase.full {
					data.scenes = append(data.scenes, fillerScenes()...)
				}

				insertFixtures(ctx, t, data)

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
