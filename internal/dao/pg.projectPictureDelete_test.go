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

func TestPgProjectPictureDelete(t *testing.T) {
	t.Parallel()

	fixtures := twoPictureFixtures()
	storedLogo, storedBackground := fixtures.pictures[0], fixtures.pictures[1]

	// withoutLogo is the test project once its logo is gone, its background version kept.
	withoutLogo := twoPictureFixtures().projects[0]
	withoutLogo.LogoUpdatedAt = nil

	testCases := []struct {
		name string

		request *dao.ProjectPictureDeleteRequest

		expect *dao.Project
		// expectPictures are the logo and the background of the test project after the write.
		expectPictures [2]*dao.ProjectPicture
		expectErr      error
	}{
		{
			name: "Success",

			request: &dao.ProjectPictureDeleteRequest{ProjectID: projectID, Picture: "logo"},

			expect:         withoutLogo,
			expectPictures: [2]*dao.ProjectPicture{nil, storedBackground},
		},
		{
			name: "Success/NoPicture",

			request: &dao.ProjectPictureDeleteRequest{ProjectID: otherProjectID, Picture: "logo"},

			expect:         fixtures.projects[1],
			expectPictures: [2]*dao.ProjectPicture{storedLogo, storedBackground},
		},
		{
			name: "Error/ProjectNotFound",

			request: &dao.ProjectPictureDeleteRequest{ProjectID: absentID, Picture: "logo"},

			expectPictures: [2]*dao.ProjectPicture{storedLogo, storedBackground},
			expectErr:      sql.ErrNoRows,
		},
	}

	daoProjectPictureDelete := dao.NewPgProjectPictureDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, twoPictureFixtures())

				result, err := daoProjectPictureDelete.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
				require.Equal(t, testCase.expectPictures, projectPictures(ctx, t))
			})
		})
	}
}
