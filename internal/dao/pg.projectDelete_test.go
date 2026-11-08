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

		fixtures fixtures
		request  *dao.ProjectDeleteRequest

		expect       *dao.Project
		expectScenes []*dao.Scene
		// expectPictures are the logo and the background of the test project after the delete.
		expectPictures [2]*dao.ProjectPicture
		expectErr      error
	}{
		{
			name: "Success",

			fixtures: composerFixtures(),
			request:  &dao.ProjectDeleteRequest{ID: projectID},

			expect:       composerFixtures().projects[0],
			expectScenes: []*dao.Scene{},
		},
		{
			// The project returns the versions it held, and its pictures go with it.
			name: "Success/Pictures",

			fixtures: twoPictureFixtures(),
			request:  &dao.ProjectDeleteRequest{ID: projectID},

			expect:       twoPictureFixtures().projects[0],
			expectScenes: []*dao.Scene{},
		},
		{
			name: "Error/NotFound",

			fixtures: twoPictureFixtures(),
			request:  &dao.ProjectDeleteRequest{ID: absentID},

			expectScenes:   composerFixtures().scenes[:4],
			expectPictures: [2]*dao.ProjectPicture{twoPictureFixtures().pictures[0], twoPictureFixtures().pictures[1]},
			expectErr:      dao.ErrProjectDeleteNotFound,
		},
	}

	daoProjectDelete := dao.NewPgProjectDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, testCase.fixtures)

				result, err := daoProjectDelete.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				// Deleting a project deletes its scenes with it.
				require.Equal(t, testCase.expectScenes, listScenes(ctx, t, projectID))
				require.Equal(t, testCase.expectPictures, projectPictures(ctx, t))
			})
		})
	}
}
