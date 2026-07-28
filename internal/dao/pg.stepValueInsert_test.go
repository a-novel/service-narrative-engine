package dao_test

import (
	"context"
	"encoding/json"
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

	now := time.Date(2026, 7, 28, 1, 2, 3, 123456000, time.UTC)
	testCases := []struct {
		name string

		request   *dao.StepValueInsertRequest
		insertOne bool

		expect    *dao.StepValue
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.StepValueInsertRequest{
				ID:              uuid.MustParse("00000000-0000-0000-0000-000000000301"),
				IdeaID:          fixtureIdeaID,
				EngineVersionID: fixtureEngineVersionID,
				StepKey:         "manuscript",
				Value:           fixtureManuscriptValue,
				Now:             now,
			},

			expect: &dao.StepValue{
				ID:              uuid.MustParse("00000000-0000-0000-0000-000000000301"),
				IdeaID:          fixtureIdeaID,
				EngineVersionID: fixtureEngineVersionID,
				StepKey:         "manuscript",
				CreatedAt:       now,
			},
		},
		{
			name: "Error/Conflict",

			request: &dao.StepValueInsertRequest{
				ID:              uuid.MustParse("00000000-0000-0000-0000-000000000302"),
				IdeaID:          fixtureIdeaID,
				EngineVersionID: fixtureEngineVersionID,
				StepKey:         "manuscript",
				Value:           fixtureManuscriptValue,
				Now:             now,
			},
			insertOne: true,

			expectErr: dao.ErrStepValueInsertConflict,
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

					if testCase.insertOne {
						_, err := operation.Exec(ctx, &dao.StepValueInsertRequest{
							ID:              uuid.MustParse("00000000-0000-0000-0000-000000000399"),
							IdeaID:          testCase.request.IdeaID,
							EngineVersionID: testCase.request.EngineVersionID,
							StepKey:         testCase.request.StepKey,
							Value:           json.RawMessage(`{"title":"First"}`),
							Now:             now.Add(-time.Second),
						})
						require.NoError(t, err)
					}

					stepValue, err := operation.Exec(ctx, testCase.request)
					require.ErrorIs(t, err, testCase.expectErr)

					if stepValue != nil {
						require.JSONEq(t, string(testCase.request.Value), string(stepValue.Value))
						stepValue.Value = nil
					}

					require.Equal(t, testCase.expect, stepValue)
				},
			)
		})
	}
}
