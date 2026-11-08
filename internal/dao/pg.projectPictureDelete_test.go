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

	fixtures := pictureFixtures()
	storedLogo := &dao.ProjectPicture{
		ProjectID: projectID, Picture: "logo", Data: []byte("stored logo"), MediaType: "image/webp",
		UpdatedAt: updatedAt,
	}

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

			expect: composerFixtures().projects[0],
		},
		{
			name: "Success/NoPicture",

			request: &dao.ProjectPictureDeleteRequest{ProjectID: projectID, Picture: "background"},

			expect:         fixtures.projects[0],
			expectPictures: [2]*dao.ProjectPicture{storedLogo, nil},
		},
		{
			name: "Error/ProjectNotFound",

			request: &dao.ProjectPictureDeleteRequest{ProjectID: absentID, Picture: "logo"},

			expectPictures: [2]*dao.ProjectPicture{storedLogo, nil},
			expectErr:      sql.ErrNoRows,
		},
	}

	daoProjectPictureDelete := dao.NewPgProjectPictureDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, pictureFixtures())

				result, err := daoProjectPictureDelete.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
				require.Equal(t, testCase.expectPictures, projectPictures(ctx, t))
			})
		})
	}
}
