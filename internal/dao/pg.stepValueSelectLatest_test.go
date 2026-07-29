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

func TestPgStepValueSelectLatest(t *testing.T) {
	t.Parallel()

	secondEngineVersionID := uuid.MustParse("00000000-0000-0000-0000-000000000101")
	now := time.Date(2026, 7, 29, 1, 2, 3, 123456000, time.UTC)
	charactersLatest := &dao.StepValue{
		ID:              uuid.MustParse("00000000-0000-0000-0000-000000000603"),
		IdeaID:          fixtureIdeaID,
		EngineVersionID: secondEngineVersionID,
		StepKey:         "characters",
		Value:           json.RawMessage(`{"names":["Mara","Ilan"]}`),
		CreatedAt:       now.Add(2 * time.Second),
	}
	outline := &dao.StepValue{
		ID:              uuid.MustParse("00000000-0000-0000-0000-000000000602"),
		IdeaID:          fixtureIdeaID,
		EngineVersionID: fixtureEngineVersionID,
		StepKey:         "outline",
		Value:           json.RawMessage(`{"beats":["The foghorn answers."]}`),
		CreatedAt:       now.Add(time.Second),
	}

	testCases := []struct {
		name string

		exclude []string

		expect []*dao.StepValue
	}{
		{
			name:    "LatestAcrossEngineVersions",
			exclude: []string{},
			expect: []*dao.StepValue{
				charactersLatest,
				outline,
			},
		},
		{
			name:    "ExcludeOverrideFromStorage",
			exclude: []string{"characters"},
			expect:  []*dao.StepValue{outline},
		},
		{
			name:    "ExcludeEverySavedKey",
			exclude: []string{"characters", "outline"},
			expect:  []*dao.StepValue{},
		},
	}

	operation := dao.NewPgStepValueSelectLatest()

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

					db, err := postgres.GetContext(ctx)
					require.NoError(t, err)

					_, err = db.NewInsert().Model(&dao.EngineVersion{
						ID:         secondEngineVersionID,
						EngineID:   fixtureEngineID,
						Version:    "0.0.2",
						Definition: fixtureEngineDefinition,
						CreatedAt:  now,
					}).Exec(ctx)
					require.NoError(t, err)

					stepValues := []*dao.StepValue{
						{
							ID:              uuid.MustParse("00000000-0000-0000-0000-000000000601"),
							IdeaID:          fixtureIdeaID,
							EngineVersionID: fixtureEngineVersionID,
							StepKey:         "characters",
							Value:           json.RawMessage(`{"names":["Mara"]}`),
							CreatedAt:       now,
						},
						outline,
						charactersLatest,
					}

					_, err = db.NewInsert().Model(&stepValues).Exec(ctx)
					require.NoError(t, err)

					result, err := operation.Exec(ctx, &dao.StepValueSelectLatestRequest{
						IdeaID:          fixtureIdeaID,
						ExcludeStepKeys: testCase.exclude,
					})
					require.NoError(t, err)
					require.Len(t, result, len(testCase.expect))

					for index := range result {
						require.JSONEq(
							t,
							string(testCase.expect[index].Value),
							string(result[index].Value),
						)

						result[index].Value = nil
						expected := *testCase.expect[index]
						expected.Value = nil
						require.Equal(t, &expected, result[index])
					}
				},
			)
		})
	}
}
