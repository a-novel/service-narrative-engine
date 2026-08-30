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

func TestPgManuscriptList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		seedCount int

		expectCount int
		expectFirst string
		expectLast  string
	}{
		{
			name:        "Success/RetainedNewest25",
			seedCount:   30,
			expectCount: 25,
			expectFirst: `{"blocks":[{"version":29}]}`,
			expectLast:  `{"blocks":[{"version":5}]}`,
		},
		{
			name: "Success/Empty",
		},
	}

	operation := dao.NewPgManuscriptList()

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
						insertManuscriptHistoryFixtures(t, ctx, testCase.seedCount)
					}

					manuscripts, err := operation.Exec(ctx, &dao.ManuscriptListRequest{
						ProjectID: fixtureProjectID,
					})
					require.NoError(t, err)
					require.Len(t, manuscripts, testCase.expectCount)

					if testCase.expectCount > 0 {
						require.JSONEq(t, testCase.expectFirst, string(manuscripts[0].Value))
						require.JSONEq(t, testCase.expectLast, string(manuscripts[len(manuscripts)-1].Value))
					}
				},
			)
		})
	}
}
