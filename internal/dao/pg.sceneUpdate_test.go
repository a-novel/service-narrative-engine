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

func TestPgSceneUpdate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.SceneUpdateRequest

		expect    *dao.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.SceneUpdateRequest{
				ID: sceneID2, Title: "Le mensonge", Color: "#e0457b", Description: "Paul ment.",
				Prompt: "Garder le suspense.", Now: now,
			},

			expect: &dao.Scene{
				ID: sceneID2, ProjectID: projectID, Position: 1, Title: "Le mensonge", Color: "#e0457b",
				Description: "Paul ment.", Prompt: "Garder le suspense.", UpdatedAt: now,
			},
		},
		{
			name: "Error/NotFound",

			request: &dao.SceneUpdateRequest{
				ID: absentID, Title: "Le mensonge", Color: "#e0457b", Now: now,
			},

			expectErr: dao.ErrSceneUpdateNotFound,
		},
	}

	daoSceneUpdate := dao.NewPgSceneUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoSceneUpdate.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
