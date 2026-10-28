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

		expect bool
	}{
		{
			name: "Success",

			request: &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID},

			expect: true,
		},
		{
			name: "Success/OtherOwner",

			request: &dao.ProjectLockRequest{ID: otherProjectID, OwnerID: ownerID},

			expect: false,
		},
		{
			name: "Success/NotFound",

			request: &dao.ProjectLockRequest{ID: absentID, OwnerID: ownerID},

			expect: false,
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
				require.NoError(t, err)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
