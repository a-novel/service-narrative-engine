package dao_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgProjectLogoDelete(t *testing.T) {
	t.Parallel()

	fixtures := logoFixtures()
	storedLogo := fixtures.logos[0]

	// withoutLogo is the test project once its logo is gone.
	withoutLogo := logoFixtures().projects[0]
	withoutLogo.LogoChecksum = nil

	testCases := []struct {
		name string

		request *dao.ProjectLogoDeleteRequest

		expect *dao.Project
		// expectLogo is the logo of the test project after the write.
		expectLogo *dao.ProjectLogo
		expectErr  error
	}{
		{
			name: "Success",

			request: &dao.ProjectLogoDeleteRequest{ProjectID: projectID},

			expect: withoutLogo,
		},
		{
			name: "Success/NoLogo",

			request: &dao.ProjectLogoDeleteRequest{ProjectID: otherProjectID},

			expect:     fixtures.projects[1],
			expectLogo: storedLogo,
		},
		{
			name: "Error/ProjectNotFound",

			request: &dao.ProjectLogoDeleteRequest{ProjectID: absentID},

			expectLogo: storedLogo,
			expectErr:  sql.ErrNoRows,
		},
	}

	daoProjectLogoDelete := dao.NewPgProjectLogoDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, logoFixtures())

				result, err := daoProjectLogoDelete.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
				require.Equal(t, testCase.expectLogo, projectLogo(ctx, t))
			})
		})
	}
}
