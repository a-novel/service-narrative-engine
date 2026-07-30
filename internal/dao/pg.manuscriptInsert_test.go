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

func TestPgManuscriptInsert(t *testing.T) {
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

	operation := dao.NewPgManuscriptInsert()

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
						var latest *dao.Manuscript

						for index := 1; index <= testCase.versions; index++ {
							value := json.RawMessage(fmt.Sprintf(`{"revision":%d}`, index))

							var err error

							latest, err = operation.Exec(ctx, &dao.ManuscriptInsertRequest{
								ID: uuid.MustParse(fmt.Sprintf(
									"00000000-0000-0000-0000-%012d",
									400+index,
								)),
								IdeaID:  fixtureIdeaID,
								OwnerID: fixtureOwnerID,
								Value:   value,
								Now: fixtureCreatedAt.Add(
									time.Duration(index) * time.Second,
								),
							})
							require.NoError(t, err)
						}

						db, err := postgres.GetContext(ctx)
						require.NoError(t, err)

						var manuscripts []*dao.Manuscript

						err = db.NewSelect().
							Model(&manuscripts).
							Where("idea_id = ?", fixtureIdeaID).
							OrderExpr("created_at DESC, id DESC").
							Scan(ctx)
						require.NoError(t, err)
						require.Len(t, manuscripts, min(testCase.versions, 25))

						latestWithoutValue := *latest
						selectedWithoutValue := *manuscripts[0]
						latestWithoutValue.Value = nil
						selectedWithoutValue.Value = nil
						require.Equal(t, latestWithoutValue, selectedWithoutValue)
						require.JSONEq(t, string(latest.Value), string(manuscripts[0].Value))

						return nil
					})
					require.NoError(t, err)
				},
			)
		})
	}
}
