package dao_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgSceneInsert(t *testing.T) {
	t.Parallel()

	newSceneID := uuid.MustParse("00000000-0000-0000-0000-000000000014")

	testCases := []struct {
		name string

		request *dao.SceneInsertRequest

		expect *dao.Scene
		// expectScenes is the first beat's order after the insert, when the case checks it.
		expectScenes []*dao.Scene
		// full fills the first beat up to its last position before the insert.
		full      bool
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.SceneInsertRequest{
				ID: newSceneID, ProjectID: projectID, BeatID: beatID1, Position: 3, Title: "Le débat",
				Description: "Marie hésite.", Prompt: "Lent.", Now: now,
			},

			expect: &dao.Scene{
				ID: newSceneID, ProjectID: projectID, BeatID: beatID1, Position: 3, Title: "Le débat",
				Description: "Marie hésite.", Prompt: "Lent.", UpdatedAt: now,
			},
		},
		{
			name: "Success/PastTheEnd",

			request: &dao.SceneInsertRequest{
				ID: newSceneID, ProjectID: projectID, BeatID: beatID1, Position: 9, Title: "Le débat", Now: now,
			},

			expect: &dao.Scene{
				ID: newSceneID, ProjectID: projectID, BeatID: beatID1, Position: 3, Title: "Le débat", UpdatedAt: now,
			},
		},
		{
			name: "Error/BeatFull",

			request: &dao.SceneInsertRequest{
				ID: newSceneID, ProjectID: projectID, BeatID: beatID1, Position: 0, Title: "Le débat", Now: now,
			},

			full:      true,
			expectErr: dao.ErrSceneInsertFull,
		},
		{
			name: "Error/BeatOfOtherProject",

			request: &dao.SceneInsertRequest{
				ID: newSceneID, ProjectID: projectID, BeatID: otherBeatID, Position: 0, Title: "Le débat", Now: now,
			},

			expectErr: dao.ErrSceneInsertBeatNotFound,
		},
		{
			name: "Success/Shift",

			request: &dao.SceneInsertRequest{
				ID: newSceneID, ProjectID: projectID, BeatID: beatID1, Position: 1, Title: "Le débat", Now: now,
			},

			expect: &dao.Scene{
				ID: newSceneID, ProjectID: projectID, BeatID: beatID1, Position: 1, Title: "Le débat", UpdatedAt: now,
			},
			expectScenes: []*dao.Scene{
				composerFixtures().scenes[0],
				{ID: newSceneID, ProjectID: projectID, BeatID: beatID1, Position: 1, Title: "Le débat", UpdatedAt: now},
				sceneAt(composerFixtures().scenes[1], 2),
				sceneAt(composerFixtures().scenes[2], 3),
			},
		},
	}

	daoSceneInsert := dao.NewPgSceneInsert()

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

				result, err := daoSceneInsert.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				if testCase.expectScenes != nil {
					require.Equal(t, testCase.expectScenes, beatScenes(ctx, t, beatID1))
				}
			})
		})
	}
}
