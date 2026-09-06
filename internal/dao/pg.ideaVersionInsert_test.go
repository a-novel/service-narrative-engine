package dao_test

import (
	"context"
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

func TestPgIdeaVersionInsert(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		ownerID   uuid.UUID
		expectErr error
	}{
		{
			name:    "RetainsNewest25",
			ownerID: fixtureOwnerID,
		},
		{
			name:      "OtherOwner",
			ownerID:   uuid.MustParse("00000000-0000-0000-0000-000000000043"),
			expectErr: dao.ErrProjectLockNotFound,
		},
	}

	operation := dao.NewPgIdeaVersionInsert()

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
						var latest *dao.IdeaVersion

						for index := 1; index <= 25; index++ {
							versionID := uuid.MustParse(fmt.Sprintf(
								"00000000-0000-0000-0000-%012d",
								800+index,
							))

							version, err := operation.Exec(ctx, &dao.IdeaVersionInsertRequest{
								ID:        versionID,
								ProjectID: fixtureProjectID,
								OwnerID:   testCase.ownerID,
								Seed:      fmt.Sprintf("Revision %d", index),
								Genre:     "speculative",
								Title:     "The Answering Light",
								Now: fixtureCreatedAt.Add(
									time.Duration(index) * time.Second,
								),
							})
							if err != nil {
								return err
							}

							latest = version
						}

						db, err := postgres.GetContext(ctx)
						require.NoError(t, err)

						var versions []*dao.IdeaVersion

						err = db.NewSelect().
							Model(&versions).
							Where("project_id = ?", fixtureProjectID).
							OrderExpr("created_at DESC, id DESC").
							Scan(ctx)
						require.NoError(t, err)
						require.Len(t, versions, 25)
						require.Equal(t, latest, versions[0])
						require.NotEqual(t, fixtureIdeaVersionID, versions[len(versions)-1].ID)

						return nil
					})
					require.ErrorIs(t, err, testCase.expectErr)
				},
			)
		})
	}
}
