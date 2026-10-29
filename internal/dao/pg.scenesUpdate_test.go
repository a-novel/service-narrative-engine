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

func TestPgScenesUpdate(t *testing.T) {
	t.Parallel()

	scenes := composerFixtures().scenes

	// updateScene2 and updateScene3 rewrite the second and third scenes, which become scene2 and scene3.
	updateScene2 := dao.ScenesUpdateItem{
		ID: sceneID2, Title: "Le mensonge", Description: "Paul ment.", Prompt: "Garder le suspense.",
	}
	updateScene3 := dao.ScenesUpdateItem{ID: sceneID3, Title: "Le départ"}
	scene2 := &dao.Scene{
		ID: sceneID2, ProjectID: projectID, BeatID: beatID1, Position: 1, Title: "Le mensonge",
		Description: "Paul ment.", Prompt: "Garder le suspense.", UpdatedAt: now,
	}
	scene3 := &dao.Scene{
		ID: sceneID3, ProjectID: projectID, BeatID: beatID1, Position: 2, Title: "Le départ", UpdatedAt: now,
	}

	testCases := []struct {
		name string

		request *dao.ScenesUpdateRequest

		expect []*dao.Scene
		// expectScenes is the project's order after the update.
		expectScenes []*dao.Scene
	}{
		{
			name: "Success",

			request: &dao.ScenesUpdateRequest{ProjectID: projectID, Scenes: []dao.ScenesUpdateItem{updateScene2}, Now: now},

			expect:       []*dao.Scene{scene2},
			expectScenes: []*dao.Scene{scenes[0], scene2, scenes[2], scenes[3]},
		},
		{
			name: "Success/Batch",

			request: &dao.ScenesUpdateRequest{
				ProjectID: projectID, Scenes: []dao.ScenesUpdateItem{updateScene2, updateScene3}, Now: now,
			},

			expect:       []*dao.Scene{scene2, scene3},
			expectScenes: []*dao.Scene{scenes[0], scene2, scene3, scenes[3]},
		},
		{
			name: "Success/NotFound",

			request: &dao.ScenesUpdateRequest{
				ProjectID: projectID,
				Scenes:    []dao.ScenesUpdateItem{updateScene2, {ID: absentID, Title: "Le départ"}},
				Now:       now,
			},

			expectScenes: scenes[:4],
		},
		{
			name: "Success/OtherProject",

			request: &dao.ScenesUpdateRequest{
				ProjectID: projectID,
				Scenes:    []dao.ScenesUpdateItem{updateScene2, {ID: otherSceneID, Title: "Le départ"}},
				Now:       now,
			},

			expectScenes: scenes[:4],
		},
	}

	daoScenesUpdate := dao.NewPgScenesUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoScenesUpdate.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.ElementsMatch(t, testCase.expect, result)

				// An ID the project does not hold leaves every scene unchanged, the valid ones included.
				require.Equal(t, testCase.expectScenes, listScenes(ctx, t, projectID))
			})
		})
	}
}
