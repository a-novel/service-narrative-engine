package dao_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
	"github.com/a-novel/service-narrative-engine/v2/internal/models/migrations"
)

func TestPgInstanceCount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.InstanceCountRequest

		expect *dao.InstanceCount
	}{
		{
			name: "Success",

			request: &dao.InstanceCountRequest{ProjectID: projectID},

			expect: &dao.InstanceCount{Count: 2},
		},
		{
			name: "Success/OtherProject",

			request: &dao.InstanceCountRequest{ProjectID: otherProjectID},

			expect: &dao.InstanceCount{Count: 1},
		},
		{
			name: "Success/None",

			request: &dao.InstanceCountRequest{ProjectID: absentID},

			expect: &dao.InstanceCount{Count: 0},
		},
	}

	daoInstanceCount := dao.NewPgInstanceCount()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, instanceFixtures())

				result, err := daoInstanceCount.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
