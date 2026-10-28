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

func TestPgProjectLock(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.ProjectLockRequest

		expect    *dao.Project
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ProjectLockRequest{ID: projectID},

			expect: composerFixtures().projects[0],
		},
		{
			name: "Error/NotFound",

			request: &dao.ProjectLockRequest{ID: absentID},

			expectErr: dao.ErrProjectLockNotFound,
		},
	}

	daoProjectLock := dao.NewPgProjectLock()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoProjectLock.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
