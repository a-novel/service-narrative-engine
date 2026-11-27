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

func TestPgInstanceList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.InstanceListRequest

		expect []*dao.InstanceListRow
	}{
		{
			name: "Success",

			request: &dao.InstanceListRequest{ProjectID: projectID, Limit: 10},

			expect: instanceFixtures().page(2, instanceID, otherInstanceID),
		},
		{
			// The total counts every instance of the project, beyond the page.
			name: "Success/Limit",

			request: &dao.InstanceListRequest{ProjectID: projectID, Limit: 1},

			expect: instanceFixtures().page(2, instanceID),
		},
		{
			name: "Success/Offset",

			request: &dao.InstanceListRequest{ProjectID: projectID, Limit: 10, Offset: 1},

			expect: instanceFixtures().page(2, otherInstanceID),
		},
		{
			// A page past the end has no row to carry the total.
			name: "Success/PastTheEnd",

			request: &dao.InstanceListRequest{ProjectID: projectID, Limit: 10, Offset: 2},

			expect: []*dao.InstanceListRow{},
		},
		{
			name: "Success/Empty",

			request: &dao.InstanceListRequest{ProjectID: absentID, Limit: 10},

			expect: []*dao.InstanceListRow{},
		},
	}

	daoInstanceList := dao.NewPgInstanceList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, instanceFixtures())

				result, err := daoInstanceList.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
