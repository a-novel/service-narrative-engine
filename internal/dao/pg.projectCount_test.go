package dao_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgProjectCount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.ProjectCountRequest

		expect    *dao.ProjectCount
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ProjectCountRequest{OwnerID: ownerID},

			expect: &dao.ProjectCount{Count: 1},
		},
		{
			name: "Success/NoProject",

			request: &dao.ProjectCountRequest{OwnerID: uuid.New()},

			expect: &dao.ProjectCount{Count: 0},
		},
	}

	daoProjectCount := dao.NewPgProjectCount()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoProjectCount.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
