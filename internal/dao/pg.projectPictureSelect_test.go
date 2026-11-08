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

func TestPgProjectPictureSelect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.ProjectPictureSelectRequest

		expect    *dao.ProjectPicture
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ProjectPictureSelectRequest{ProjectID: projectID, Picture: "logo"},

			expect: &dao.ProjectPicture{
				ProjectID: projectID, Picture: "logo", Data: []byte("stored logo"), MediaType: "image/webp",
				UpdatedAt: updatedAt,
			},
		},
		{
			name: "Error/NoPicture",

			request: &dao.ProjectPictureSelectRequest{ProjectID: projectID, Picture: "background"},

			expectErr: dao.ErrProjectPictureSelectNotFound,
		},
		{
			name: "Error/OtherProject",

			request: &dao.ProjectPictureSelectRequest{ProjectID: otherProjectID, Picture: "logo"},

			expectErr: dao.ErrProjectPictureSelectNotFound,
		},
	}

	daoProjectPictureSelect := dao.NewPgProjectPictureSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, pictureFixtures())

				result, err := daoProjectPictureSelect.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
