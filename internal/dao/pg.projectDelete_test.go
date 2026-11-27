package dao_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
	"github.com/a-novel/service-narrative-engine/v2/internal/models/migrations"
)

func TestPgProjectDelete(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		fixtures fixtures
		request  *dao.ProjectDeleteRequest

		expect *dao.Project
		// expectLogo is the logo of the test project after the delete.
		expectLogo *dao.ProjectLogo
		// expectKept are the instances of the other project after the delete.
		expectKept []*dao.Instance
		expectErr  error
	}{
		{
			name: "Success",

			fixtures: projectFixtures(),
			request:  &dao.ProjectDeleteRequest{ID: projectID},

			expect:     projectFixtures().projects[0],
			expectKept: []*dao.Instance{},
		},
		{
			// The project returns the logo checksum it held, and its logo goes with it.
			name: "Success/Logo",

			fixtures: logoFixtures(),
			request:  &dao.ProjectDeleteRequest{ID: projectID},

			expect:     logoFixtures().projects[0],
			expectKept: []*dao.Instance{},
		},
		{
			// The delete crosses the cascade to the project's instances and the restriction on its default
			// instance, and takes the instances and the links with it.
			name: "Success/Instances",

			fixtures: instanceFixtures(),
			request:  &dao.ProjectDeleteRequest{ID: projectID},

			expect:     instanceFixtures().projects[0],
			expectKept: instanceFixtures().instances[2:],
		},
		{
			name: "Error/NotFound",

			fixtures: logoFixtures(),
			request:  &dao.ProjectDeleteRequest{ID: absentID},

			expectLogo: logoFixtures().logos[0],
			expectKept: []*dao.Instance{},
			expectErr:  dao.ErrProjectNotFound,
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

				// Deleting a project deletes its logo with it.
				require.Equal(t, testCase.expectLogo, projectLogo(ctx, t))

				// It deletes its instances too, with the links they store, and leaves those of the other project.
				require.Empty(t, projectInstances(ctx, t, projectID))
				require.Empty(t, storedInstanceLinks(ctx, t, projectID))
				require.Equal(t, testCase.expectKept, projectInstances(ctx, t, otherProjectID))
			})
		})
	}
}
