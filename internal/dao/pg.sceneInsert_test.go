package dao_test

import (
	"cmp"
	"context"
	"slices"
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

	newSceneID1 := uuid.MustParse("00000000-0000-0000-0000-000000000014")
	newSceneID2 := uuid.MustParse("00000000-0000-0000-0000-000000000015")

	testCases := []struct {
		name string

		request *dao.SceneInsertRequest

		expect []*dao.Scene
		// expectScenes is the project's order after the insert, when the case checks it.
		expectScenes []*dao.Scene
		expectErr    error
	}{
		{
			name: "Success",

			request: &dao.SceneInsertRequest{
				ProjectID: projectID,
				Position:  3,
				Scenes: []dao.SceneInsertItem{
					{ID: newSceneID1, Title: "Le débat", Color: "#e0457b", Description: "Marie hésite.", Prompt: "Lent."},
					{ID: newSceneID2, Title: "Le passage", Color: "#2bb673"},
				},
				Now: now,
			},

			expect: []*dao.Scene{
				{
					ID: newSceneID1, ProjectID: projectID, Position: 3, Title: "Le débat", Color: "#e0457b",
					Description: "Marie hésite.", Prompt: "Lent.", UpdatedAt: now,
				},
				{ID: newSceneID2, ProjectID: projectID, Position: 4, Title: "Le passage", Color: "#2bb673", UpdatedAt: now},
			},
		},
		{
			name: "Success/LastPosition",

			request: &dao.SceneInsertRequest{
				ProjectID: projectID,
				Position:  499,
				Scenes:    []dao.SceneInsertItem{{ID: newSceneID1, Title: "Le débat", Color: "#e0457b"}},
				Now:       now,
			},

			expect: []*dao.Scene{
				{ID: newSceneID1, ProjectID: projectID, Position: 499, Title: "Le débat", Color: "#e0457b", UpdatedAt: now},
			},
		},
		{
			name: "Error/ProjectFull",

			request: &dao.SceneInsertRequest{
				ProjectID: projectID,
				Position:  499,
				Scenes: []dao.SceneInsertItem{
					{ID: newSceneID1, Title: "Le débat", Color: "#e0457b"},
					{ID: newSceneID2, Title: "Le passage", Color: "#2bb673"},
				},
				Now: now,
			},

			expectErr: dao.ErrSceneInsertFull,
		},
		{
			name: "Success/Shift",

			request: &dao.SceneInsertRequest{
				ProjectID: projectID,
				Position:  1,
				Scenes: []dao.SceneInsertItem{
					{ID: newSceneID1, Title: "Le débat", Color: "#e0457b"},
					{ID: newSceneID2, Title: "Le passage", Color: "#2bb673"},
				},
				Now: now,
			},

			expect: []*dao.Scene{
				{ID: newSceneID1, ProjectID: projectID, Position: 1, Title: "Le débat", Color: "#e0457b", UpdatedAt: now},
				{ID: newSceneID2, ProjectID: projectID, Position: 2, Title: "Le passage", Color: "#2bb673", UpdatedAt: now},
			},
			expectScenes: []*dao.Scene{
				composerFixtures().scenes[0],
				{ID: newSceneID1, ProjectID: projectID, Position: 1, Title: "Le débat", Color: "#e0457b", UpdatedAt: now},
				{ID: newSceneID2, ProjectID: projectID, Position: 2, Title: "Le passage", Color: "#2bb673", UpdatedAt: now},
				sceneAt(composerFixtures().scenes[1], 3),
				sceneAt(composerFixtures().scenes[2], 4),
			},
		},
	}

	daoSceneInsert := dao.NewPgSceneInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoSceneInsert.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				slices.SortFunc(result, func(a, b *dao.Scene) int { return cmp.Compare(a.Position, b.Position) })
				require.Equal(t, testCase.expect, result)

				if testCase.expectScenes != nil {
					require.Equal(t, testCase.expectScenes, listScenes(ctx, t, projectID))
				}
			})
		})
	}
}
