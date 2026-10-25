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
				ID: sceneID2, OwnerID: ownerID, Revision: 1, Title: "Le mensonge", Synopsis: "Paul ment.", Now: now,
			},

			expect: &dao.Scene{
				ID: sceneID2, ProjectID: projectID, Position: 1, Title: "Le mensonge", Synopsis: "Paul ment.",
				Revision: 2, UpdatedAt: now,
			},
		},
		{
			name: "Error/StaleRevision",

			request: &dao.SceneUpdateRequest{ID: sceneID2, OwnerID: ownerID, Revision: 2, Title: "Le mensonge", Now: now},

			expectErr: dao.ErrSceneUpdateNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: &dao.SceneUpdateRequest{ID: sceneID2, OwnerID: otherOwnerID, Revision: 1, Title: "Le mensonge", Now: now},

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
