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

		expect       []*dao.Scene
		expectFailed bool
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
			name: "Error/PositionTaken",

			request: &dao.SceneInsertRequest{
				ProjectID: projectID,
				Position:  2,
				Scenes:    []dao.SceneInsertItem{{ID: newSceneID1, Title: "Le débat", Color: "#e0457b"}},
				Now:       now,
			},

			expectFailed: true,
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
				if testCase.expectFailed {
					require.Error(t, err)

					return
				}

				require.NoError(t, err)

				slices.SortFunc(result, func(a, b *dao.Scene) int { return cmp.Compare(a.Position, b.Position) })
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
