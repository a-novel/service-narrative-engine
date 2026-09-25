package dao_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgStepValueCurrentList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		seed bool

		expectCount int
	}{
		{
			name:        "Success/LatestPerKey",
			seed:        true,
			expectCount: 2,
		},
		{
			name: "Success/Empty",
		},
	}

	operation := dao.NewPgStepValueCurrentList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunIsolatedTransactionalTest(
				t,
				configtest.PostgresPreset,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					if testCase.seed {
						insertWalkingSkeletonFixtures(t, ctx)
						insertStepValueHistoryFixtures(t, ctx, "outline", 3)
						insertStepValueHistoryFixtures(t, ctx, "characters", 1)
					}

					values, err := operation.Exec(ctx, &dao.StepValueCurrentListRequest{
						ProjectID: fixtureProjectID,
					})
					require.NoError(t, err)
					require.Len(t, values, testCase.expectCount)

					if testCase.expectCount > 0 {
						require.Equal(t, "characters", values[0].Key)
						require.JSONEq(t, `{"version":0}`, string(values[0].Value))
						require.Equal(t, "outline", values[1].Key)
						require.JSONEq(t, `{"version":2}`, string(values[1].Value))
					}
				},
			)
		})
	}
}
