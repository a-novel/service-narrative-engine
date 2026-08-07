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

func TestPgManuscriptSelect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		seedCount int

		expectValue string
	}{
		{
			name:        "Success/Latest",
			seedCount:   3,
			expectValue: `{"blocks":[{"version":2}]}`,
		},
		{
			name: "Success/None",
		},
	}

	operation := dao.NewPgManuscriptSelect()

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

					manuscript, err := operation.Exec(ctx, &dao.ManuscriptSelectRequest{
						ProjectID: fixtureProjectID,
					})
					require.NoError(t, err)

					if testCase.expectValue == "" {
						require.Nil(t, manuscript)

						return
					}

					require.NotNil(t, manuscript)
					require.JSONEq(t, testCase.expectValue, string(manuscript.Value))
				},
			)
		})
	}
}
