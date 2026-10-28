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

func TestPgSceneSelect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.SceneSelectRequest

		expect    *dao.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.SceneSelectRequest{ID: sceneID2},

			expect: composerFixtures().scenes[1],
		},
		{
			name: "Error/NotFound",

			request: &dao.SceneSelectRequest{ID: absentID},

			expectErr: dao.ErrSceneSelectNotFound,
		},
	}

	daoSceneSelect := dao.NewPgSceneSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoSceneSelect.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
