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

func TestPgStepValueList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		seedCount int
		key       string

		expectCount int
		expectFirst string
		expectLast  string
	}{
		{
			name:        "Success/RetainedNewest25",
			seedCount:   30,
			key:         "outline",
			expectCount: 25,
			expectFirst: `{"version":29}`,
			expectLast:  `{"version":5}`,
		},
		{
			name: "Success/Empty",
			key:  "missing",
		},
	}

	operation := dao.NewPgStepValueList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunIsolatedTransactionalTest(
				t,
				configtest.PostgresPreset,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					if testCase.seedCount > 0 {
						insertWalkingSkeletonFixtures(t, ctx)
						insertStepValueHistoryFixtures(t, ctx, testCase.key, testCase.seedCount)
					}

					values, err := operation.Exec(ctx, &dao.StepValueListRequest{
						ProjectID: fixtureProjectID,
						Key:       testCase.key,
					})
					require.NoError(t, err)
					require.Len(t, values, testCase.expectCount)

					if testCase.expectCount > 0 {
						require.JSONEq(t, testCase.expectFirst, string(values[0].Value))
						require.JSONEq(t, testCase.expectLast, string(values[len(values)-1].Value))
					}
				},
			)
		})
	}
}
