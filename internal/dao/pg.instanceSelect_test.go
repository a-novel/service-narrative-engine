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

func TestPgInstanceSelect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.InstanceSelectRequest

		expect    *dao.Instance
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.InstanceSelectRequest{ProjectID: projectID, ID: instanceID},

			expect: instanceFixtures().instances[0],
		},
		{
			name: "Error/NotFound",

			request: &dao.InstanceSelectRequest{ProjectID: projectID, ID: absentID},

			expectErr: dao.ErrInstanceNotFound,
		},
		{
			name: "Error/OtherProject",

			request: &dao.InstanceSelectRequest{ProjectID: projectID, ID: alienInstanceID},

			expectErr: dao.ErrInstanceNotFound,
		},
	}

	daoInstanceSelect := dao.NewPgInstanceSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, instanceFixtures())

				result, err := daoInstanceSelect.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
