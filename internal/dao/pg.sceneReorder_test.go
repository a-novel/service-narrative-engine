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

func TestPgSceneReorder(t *testing.T) {
	t.Parallel()

	scene := func(id uuid.UUID, position int, title string) *dao.Scene {
		return &dao.Scene{ID: id, ProjectID: projectID, Position: position, Title: title, Revision: 1, UpdatedAt: updatedAt}
	}

	testCases := []struct {
		name string

		request *dao.SceneReorderRequest

		expect    []*dao.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.SceneReorderRequest{ProjectID: projectID, IDs: []uuid.UUID{sceneID3, sceneID1, sceneID2}},

			expect: []*dao.Scene{
				scene(sceneID3, 0, "Le déclencheur"),
				scene(sceneID1, 1, "Ouverture"),
				scene(sceneID2, 2, "Le thème"),
			},
		},
		{
			name: "Success/OtherProjectIgnored",

			request: &dao.SceneReorderRequest{ProjectID: projectID, IDs: []uuid.UUID{sceneID1, otherSceneID}},

			expect: []*dao.Scene{scene(sceneID1, 0, "Ouverture")},
		},
	}

	daoSceneReorder := dao.NewPgSceneReorder()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoSceneReorder.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				slices.SortFunc(result, func(a, b *dao.Scene) int { return cmp.Compare(a.Position, b.Position) })
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
