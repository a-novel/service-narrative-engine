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

func TestPgStepValueInsert(t *testing.T) {
	t.Parallel()

	otherOwnerID := uuid.MustParse("00000000-0000-0000-0000-000000000043")
	testCases := []struct {
		name string

		ownerID  uuid.UUID
		versions int
		value    json.RawMessage

		expectErr error
	}{
		{name: "Success/Object", ownerID: fixtureOwnerID, versions: 1, value: json.RawMessage(`{"freeform":true}`)},
		{name: "Success/Array", ownerID: fixtureOwnerID, versions: 1, value: json.RawMessage(`[1,"two"]`)},
		{name: "Success/String", ownerID: fixtureOwnerID, versions: 1, value: json.RawMessage(`"freeform"`)},
		{name: "Success/Number", ownerID: fixtureOwnerID, versions: 1, value: json.RawMessage(`42`)},
		{name: "Success/Boolean", ownerID: fixtureOwnerID, versions: 1, value: json.RawMessage(`true`)},
		{name: "Success/Null", ownerID: fixtureOwnerID, versions: 1, value: json.RawMessage(`null`)},
		{name: "Success/RepeatedSaveRetainsNewest25", ownerID: fixtureOwnerID, versions: 26},
		{
			name: "Error/OtherOwner", ownerID: otherOwnerID, versions: 1,
			value: json.RawMessage(`{"private":true}`), expectErr: dao.ErrProjectLockNotFound,
		},
	}

	operation := dao.NewPgStepValueInsert()

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
						_, err := operation.Exec(ctx, &dao.StepValueInsertRequest{
							ID:        uuid.MustParse("00000000-0000-0000-0000-000000000399"),
							ProjectID: fixtureProjectID,
							OwnerID:   fixtureOwnerID,
							Key:       "unrelated",
							Value:     json.RawMessage(`{"kept":true}`),
							Now:       fixtureCreatedAt,
						})
						if err != nil {
							return err
						}

						var latest *dao.StepValue

						for index := 1; index <= testCase.versions; index++ {
							value := testCase.value
							if testCase.versions > 1 {
								value = json.RawMessage(fmt.Sprintf(`{"revision":%d}`, index))
							}

							latest, err = operation.Exec(ctx, &dao.StepValueInsertRequest{
								ID: uuid.MustParse(fmt.Sprintf(
									"00000000-0000-0000-0000-%012d",
									300+index,
								)),
								ProjectID: fixtureProjectID,
								OwnerID:   testCase.ownerID,
								Key:       "client-key",
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

						var stepValues []*dao.StepValue

						err = db.NewSelect().
							Model(&stepValues).
							Where("project_id = ?", fixtureProjectID).
							Where("key = ?", "client-key").
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
							Where("project_id = ?", fixtureProjectID).
							Where("key = ?", "unrelated").
							Count(ctx)
						require.NoError(t, err)
						require.Equal(t, 1, count)

						return nil
					})
					require.ErrorIs(t, err, testCase.expectErr)
				},
			)
		})
	}
}
