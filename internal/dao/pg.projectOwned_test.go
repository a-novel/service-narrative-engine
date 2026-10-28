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

func TestPgProjectOwned(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.ProjectOwnedRequest

		expect bool
	}{
		{
			name: "Success",

			request: &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID},

			expect: true,
		},
		{
			name: "Success/OtherOwner",

			request: &dao.ProjectOwnedRequest{ID: otherProjectID, OwnerID: ownerID},

			expect: false,
		},
		{
			name: "Success/NotFound",

			request: &dao.ProjectOwnedRequest{ID: absentID, OwnerID: ownerID},

			expect: false,
		},
	}

	daoProjectOwned := dao.NewPgProjectOwned()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoProjectOwned.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
