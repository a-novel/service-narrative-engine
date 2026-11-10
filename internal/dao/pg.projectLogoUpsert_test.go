package dao_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgProjectLogoUpsert(t *testing.T) {
	t.Parallel()

	// pgUniqueViolation is the SQLSTATE of a write that breaks a unique key.
	const pgUniqueViolation = "23505"

	// withLogo returns the test project with the version of its new logo.
	withLogo := func(logo *dao.ProjectLogo) *dao.Project {
		project := composerFixtures().projects[0]
		project.LogoUpdatedAt = lo.ToPtr(logo.UpdatedAt)

		return project
	}

	storedLogo := logoFixtures().logos[0]
	newLogo := &dao.ProjectLogo{ProjectID: projectID, Data: []byte("new logo"), MediaType: "image/jpeg", UpdatedAt: now}
	pngLogo := &dao.ProjectLogo{ProjectID: projectID, Data: []byte("png logo"), MediaType: "image/png", UpdatedAt: now}

	request := func(logo *dao.ProjectLogo) *dao.ProjectLogoUpsertRequest {
		return &dao.ProjectLogoUpsertRequest{
			ProjectID: logo.ProjectID, Data: logo.Data, MediaType: logo.MediaType, Now: now,
		}
	}

	testCases := []struct {
		name string

		fixtures fixtures
		request  *dao.ProjectLogoUpsertRequest

		expect *dao.Project
		// expectLogo is the logo of the test project after the write.
		expectLogo *dao.ProjectLogo
		expectErr  bool
	}{
		{
			name: "Success/Insert",

			fixtures: composerFixtures(),
			request:  request(newLogo),

			expect:     withLogo(newLogo),
			expectLogo: newLogo,
		},
		{
			name: "Success/Replace",

			fixtures: logoFixtures(),
			request:  request(newLogo),

			expect:     withLogo(newLogo),
			expectLogo: newLogo,
		},
		{
			name: "Success/PNG",

			fixtures: logoFixtures(),
			request:  request(pngLogo),

			expect:     withLogo(pngLogo),
			expectLogo: pngLogo,
		},
		{
			name: "Error/ProjectNotFound",

			fixtures: logoFixtures(),
			request: &dao.ProjectLogoUpsertRequest{
				ProjectID: absentID, Data: newLogo.Data, MediaType: newLogo.MediaType, Now: now,
			},

			expectLogo: storedLogo,
			expectErr:  true,
		},
		{
			name: "Error/TooLarge",

			fixtures: logoFixtures(),
			request: &dao.ProjectLogoUpsertRequest{
				ProjectID: projectID, Data: bytes.Repeat([]byte{1}, 256<<10+1), MediaType: "image/webp", Now: now,
			},

			expectLogo: storedLogo,
			expectErr:  true,
		},
		{
			name: "Error/MediaType",

			fixtures: logoFixtures(),
			request: &dao.ProjectLogoUpsertRequest{
				ProjectID: projectID, Data: newLogo.Data, MediaType: "image/gif", Now: now,
			},

			expectLogo: storedLogo,
			expectErr:  true,
		},
	}

	daoProjectLogoUpsert := dao.NewPgProjectLogoUpsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, testCase.fixtures)

				result, err := daoProjectLogoUpsert.Exec(ctx, testCase.request)
				require.Equal(t, testCase.expectErr, err != nil, err)
				require.Equal(t, testCase.expect, result)
				require.Equal(t, testCase.expectLogo, projectLogo(ctx, t))
			})
		})
	}

	// The key holds one logo per project, and the upsert replaces a logo through it. A plain insert of a second
	// logo breaks it.
	t.Run("Error/DuplicateLogo", func(t *testing.T) {
		t.Parallel()

		postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
			t.Helper()

			insertFixtures(ctx, t, logoFixtures())

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			_, err = db.NewInsert().Model(newLogo).Exec(ctx)

			var pgErr pgdriver.Error

			require.ErrorAs(t, err, &pgErr)
			require.Equal(t, pgUniqueViolation, pgErr.Field('C'))
			require.Equal(t, "project_logos_pkey", pgErr.Field('n'))
			require.Equal(t, storedLogo, projectLogo(ctx, t))
		})
	})
}
