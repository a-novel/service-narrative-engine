package dao_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgStepValueInsert(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		versions int
	}{
		{
			name:     "Success",
			versions: 1,
		},
		{
			name:     "RepeatedSaveRetainsNewest25",
			versions: 26,
		},
	}

	operation := dao.NewPgStepValueInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunIsolatedTransactionalTest(
				t,
				configtest.PostgresPreset,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					insertWalkingSkeletonFixtures(t, ctx)

					err := postgres.WithinTx(ctx, nil, func(ctx context.Context) error {
						_, err := operation.Exec(ctx, &dao.StepValueInsertRequest{
							ID:              uuid.MustParse("00000000-0000-0000-0000-000000000399"),
							IdeaID:          fixtureIdeaID,
							OwnerID:         fixtureOwnerID,
							EngineVersionID: fixtureEngineVersionID,
							StepKey:         "outline",
							Value:           json.RawMessage(`{"beats":[]}`),
							Now:             fixtureCreatedAt,
						})
						require.NoError(t, err)

						var latest *dao.StepValue

						for index := 1; index <= testCase.versions; index++ {
							value := json.RawMessage(fmt.Sprintf(`{"revision":%d}`, index))
							latest, err = operation.Exec(ctx, &dao.StepValueInsertRequest{
								ID: uuid.MustParse(fmt.Sprintf(
									"00000000-0000-0000-0000-%012d",
									300+index,
								)),
								IdeaID:          fixtureIdeaID,
								OwnerID:         fixtureOwnerID,
								EngineVersionID: fixtureEngineVersionID,
								StepKey:         "manuscript",
								Value:           value,
								Now:             fixtureCreatedAt.Add(time.Duration(index) * time.Second),
							})
							require.NoError(t, err)
						}

						db, err := postgres.GetContext(ctx)
						require.NoError(t, err)

						var stepValues []*dao.StepValue

						err = db.NewSelect().
							Model(&stepValues).
							Where("idea_id = ?", fixtureIdeaID).
							Where("step_key = ?", "manuscript").
							OrderExpr("created_at DESC, id DESC").
							Scan(ctx)
						require.NoError(t, err)
						require.Len(t, stepValues, min(testCase.versions, 25))

						latestWithoutValue := *latest
						selectedWithoutValue := *stepValues[0]
						latestWithoutValue.Value = nil
						selectedWithoutValue.Value = nil
						require.Equal(t, latestWithoutValue, selectedWithoutValue)
						require.JSONEq(t, string(latest.Value), string(stepValues[0].Value))

						count, err := db.NewSelect().
							Model((*dao.StepValue)(nil)).
							Where("idea_id = ?", fixtureIdeaID).
							Where("step_key = ?", "outline").
							Count(ctx)
						require.NoError(t, err)
						require.Equal(t, 1, count)

						return nil
					})
					require.NoError(t, err)
				},
			)
		})
	}
}
