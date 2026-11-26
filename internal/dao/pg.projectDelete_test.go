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
		expectErr  error
	}{
		{
			name: "Success",

			fixtures: projectFixtures(),
			request:  &dao.ProjectDeleteRequest{ID: projectID},

			expect: projectFixtures().projects[0],
		},
		{
			// The project returns the logo checksum it held, and its logo goes with it.
			name: "Success/Logo",

			fixtures: logoFixtures(),
			request:  &dao.ProjectDeleteRequest{ID: projectID},

			expect: logoFixtures().projects[0],
		},
		{
			name: "Error/NotFound",

			fixtures: logoFixtures(),
			request:  &dao.ProjectDeleteRequest{ID: absentID},

			expectLogo: logoFixtures().logos[0],
			expectErr:  dao.ErrProjectDeleteNotFound,
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
			})
		})
	}
}
