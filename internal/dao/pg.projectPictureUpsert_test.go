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

func TestPgProjectPictureUpsert(t *testing.T) {
	t.Parallel()

	// pgUniqueViolation is the SQLSTATE of a write that breaks a unique key.
	const pgUniqueViolation = "23505"

	// withVersions returns the test project with the versions of its logo and its background.
	withVersions := func(logo, background *dao.ProjectPicture) *dao.Project {
		project := pictureFixtures().projects[0]
		project.LogoUpdatedAt = nil

		if logo != nil {
			project.LogoUpdatedAt = lo.ToPtr(logo.UpdatedAt)
		}

		if background != nil {
			project.BackgroundUpdatedAt = lo.ToPtr(background.UpdatedAt)
		}

		return project
	}

	storedLogo := &dao.ProjectPicture{
		ProjectID: projectID, Picture: "logo", Data: []byte("stored logo"), MediaType: "image/webp",
		UpdatedAt: updatedAt,
	}
	newLogo := &dao.ProjectPicture{
		ProjectID: projectID, Picture: "logo", Data: []byte("new logo"), MediaType: "image/jpeg", UpdatedAt: now,
	}
	// The background is larger than a logo may be, which only a background holds.
	newBackground := &dao.ProjectPicture{
		ProjectID: projectID, Picture: "background", Data: bytes.Repeat([]byte{1}, 512<<10), MediaType: "image/webp",
		UpdatedAt: now,
	}

	request := func(picture *dao.ProjectPicture) *dao.ProjectPictureUpsertRequest {
		return &dao.ProjectPictureUpsertRequest{
			ProjectID: picture.ProjectID, Picture: picture.Picture, Data: picture.Data, MediaType: picture.MediaType,
			Now: now,
		}
	}

	testCases := []struct {
		name string

		// stored is a picture written before the request.
		stored  *dao.ProjectPictureUpsertRequest
		request *dao.ProjectPictureUpsertRequest

		expect *dao.Project
		// expectPictures are the logo and the background of the test project after the write.
		expectPictures [2]*dao.ProjectPicture
		expectErr      bool
	}{
		{
			name: "Success/Replace",

			request: request(newLogo),

			expect:         withVersions(newLogo, nil),
			expectPictures: [2]*dao.ProjectPicture{newLogo, nil},
		},
		{
			name: "Success/Insert",

			request: request(newBackground),

			expect:         withVersions(storedLogo, newBackground),
			expectPictures: [2]*dao.ProjectPicture{storedLogo, newBackground},
		},
		{
			name: "Success/ReplaceBesideBackground",

			stored:  request(newBackground),
			request: request(newLogo),

			expect:         withVersions(newLogo, newBackground),
			expectPictures: [2]*dao.ProjectPicture{newLogo, newBackground},
		},
		{
			name: "Error/ProjectNotFound",

			request: &dao.ProjectPictureUpsertRequest{
				ProjectID: absentID, Picture: "logo", Data: newLogo.Data, MediaType: newLogo.MediaType, Now: now,
			},

			expectPictures: [2]*dao.ProjectPicture{storedLogo, nil},
			expectErr:      true,
		},
		{
			name: "Error/LogoTooLarge",

			request: &dao.ProjectPictureUpsertRequest{
				ProjectID: projectID, Picture: "logo", Data: bytes.Repeat([]byte{1}, 256<<10+1), MediaType: "image/webp",
				Now: now,
			},

			expectPictures: [2]*dao.ProjectPicture{storedLogo, nil},
			expectErr:      true,
		},
		{
			name: "Error/MediaType",

			request: &dao.ProjectPictureUpsertRequest{
				ProjectID: projectID, Picture: "logo", Data: newLogo.Data, MediaType: "image/png", Now: now,
			},

			expectPictures: [2]*dao.ProjectPicture{storedLogo, nil},
			expectErr:      true,
		},
	}

	daoProjectPictureUpsert := dao.NewPgProjectPictureUpsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, pictureFixtures())

				if testCase.stored != nil {
					_, err := daoProjectPictureUpsert.Exec(ctx, testCase.stored)
					require.NoError(t, err)
				}

				result, err := daoProjectPictureUpsert.Exec(ctx, testCase.request)
				require.Equal(t, testCase.expectErr, err != nil, err)
				require.Equal(t, testCase.expect, result)
				require.Equal(t, testCase.expectPictures, projectPictures(ctx, t))
			})
		})
	}

	// The key holds one picture of each kind per project, and the upsert replaces a picture through it. A plain
	// insert of a second logo breaks it.
	t.Run("Error/DuplicatePicture", func(t *testing.T) {
		t.Parallel()

		postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
			t.Helper()

			insertFixtures(ctx, t, pictureFixtures())

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			_, err = db.NewInsert().Model(&dao.ProjectPicture{
				ProjectID: projectID, Picture: "logo", Data: newLogo.Data, MediaType: newLogo.MediaType,
			}).Exec(ctx)

			var pgErr pgdriver.Error

			require.ErrorAs(t, err, &pgErr)
			require.Equal(t, pgUniqueViolation, pgErr.Field('C'))
			require.Equal(t, "project_pictures_pkey", pgErr.Field('n'))
			require.Equal(t, [2]*dao.ProjectPicture{storedLogo, nil}, projectPictures(ctx, t))
		})
	})
}
