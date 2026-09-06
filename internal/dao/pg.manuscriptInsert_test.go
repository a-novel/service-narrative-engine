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
	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgManuscriptInsert(t *testing.T) {
	t.Parallel()

	otherOwnerID := uuid.MustParse("00000000-0000-0000-0000-000000000043")
	testCases := []struct {
		name string

		ownerID   uuid.UUID
		versions  int
		expectErr error
	}{
		{name: "Success", ownerID: fixtureOwnerID, versions: 1},
		{name: "Success/RepeatedSaveRetainsNewest25", ownerID: fixtureOwnerID, versions: 26},
		{
			name: "Error/OtherOwner", ownerID: otherOwnerID, versions: 1,
			expectErr: dao.ErrProjectLockNotFound,
		},
	}

	operation := dao.NewPgManuscriptInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunIsolatedTransactionalTest(
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
								ProjectID: fixtureProjectID,
								OwnerID:   testCase.ownerID,
								Value:     value,
								Now: fixtureCreatedAt.Add(
									time.Duration(index) * time.Second,
								),
							})
							if err != nil {
								return err
							}
						}

						db, err := postgres.GetContext(ctx)
						require.NoError(t, err)

						var manuscripts []*dao.Manuscript

						err = db.NewSelect().
							Model(&manuscripts).
							Where("project_id = ?", fixtureProjectID).
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
					require.ErrorIs(t, err, testCase.expectErr)
				},
			)
		})
	}
}
