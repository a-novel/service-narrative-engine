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

func TestPgProjectDelete(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.ProjectDeleteRequest

		expect       *dao.Project
		expectScenes []*dao.Scene
		expectErr    error
	}{
		{
			name: "Success",

			request: &dao.ProjectDeleteRequest{ID: projectID, OwnerID: ownerID, Revision: 1},

			expect: composerFixtures().projects[0],
		},
		{
			name: "Error/StaleRevision",

			request: &dao.ProjectDeleteRequest{ID: projectID, OwnerID: ownerID, Revision: 2},

			expectScenes: composerFixtures().scenes[:3],
			expectErr:    dao.ErrProjectDeleteNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: &dao.ProjectDeleteRequest{ID: projectID, OwnerID: otherOwnerID, Revision: 1},

			expectScenes: composerFixtures().scenes[:3],
			expectErr:    dao.ErrProjectDeleteNotFound,
		},
	}

	daoProjectDelete := dao.NewPgProjectDelete()
	daoSceneList := dao.NewPgSceneList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoProjectDelete.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				// Deleting a project deletes its scenes with it.
				scenes, err := daoSceneList.Exec(ctx, &dao.SceneListRequest{ProjectID: projectID, Limit: 10})
				require.NoError(t, err)
				require.Equal(t, testCase.expectScenes, scenes)
			})
		})
	}
}
