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

func TestPgProjectSelect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.ProjectSelectRequest

		expect    *dao.Project
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ProjectSelectRequest{ID: projectID},

			expect: composerFixtures().projects[0],
		},
		{
			name: "Error/NotFound",

			request: &dao.ProjectSelectRequest{ID: absentID},

			expectErr: dao.ErrProjectSelectNotFound,
		},
	}

	daoProjectSelect := dao.NewPgProjectSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoProjectSelect.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
