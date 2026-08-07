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

func TestPgIdeaVersionList(t *testing.T) {
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
			expectFirst: "Title 29",
			expectLast:  "Title 05",
		},
		{
			name: "Success/Empty",
		},
	}

	operation := dao.NewPgIdeaVersionList()

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
						insertIdeaVersionHistoryFixtures(t, ctx, testCase.seedCount)
					}

					versions, err := operation.Exec(ctx, &dao.IdeaVersionListRequest{
						ProjectID: fixtureProjectID,
					})
					require.NoError(t, err)
					require.Len(t, versions, testCase.expectCount)

					if testCase.expectCount > 0 {
						require.Equal(t, testCase.expectFirst, versions[0].Title)
						require.Equal(t, testCase.expectLast, versions[len(versions)-1].Title)
					}
				},
			)
		})
	}
}
