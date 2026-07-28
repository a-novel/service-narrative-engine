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

// TestContentSaveRollback has no production file of its own: the property it
// proves — that a failed Manuscript insert discards the step value written
// beside it — spans two operations and belongs to neither.
func TestContentSaveRollback(t *testing.T) {
	t.Parallel()

	stepValueID := uuid.MustParse("00000000-0000-0000-0000-000000000501")
	manuscriptID := uuid.MustParse("00000000-0000-0000-0000-000000000502")
	now := time.Date(2026, 7, 28, 1, 2, 3, 123456000, time.UTC)

	postgres.RunDBTest(
		t,
		configtest.PostgresPreset,
		migrations.Migrations,
		func(ctx context.Context, t *testing.T) {
			t.Helper()

			insertWalkingSkeletonFixtures(t, ctx)

			err := postgres.NewTransactor(nil).WithinTx(ctx, func(ctx context.Context) error {
				_, err := dao.NewPgStepValueInsert().Exec(ctx, &dao.StepValueInsertRequest{
					ID:              stepValueID,
					IdeaID:          fixtureIdeaID,
					EngineVersionID: fixtureEngineVersionID,
					StepKey:         "manuscript",
					Value:           fixtureManuscriptValue,
					Now:             now,
				})
				if err != nil {
					return fmt.Errorf("insert step value: %w", err)
				}

				_, err = dao.NewPgManuscriptInsert().Exec(ctx, &dao.ManuscriptInsertRequest{
					ID:     manuscriptID,
					IdeaID: fixtureIdeaID,
					// A scalar violates manuscripts_value_check after the
					// step-value insert has already succeeded.
					Value: json.RawMessage(`"not an object"`),
					Now:   now,
				})
				if err != nil {
					return fmt.Errorf("insert manuscript: %w", err)
				}

				return nil
			})
			require.Error(t, err)

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			stepValueCount, err := db.NewSelect().
				Model((*dao.StepValue)(nil)).
				Where("id = ?", stepValueID).
				Count(ctx)
			require.NoError(t, err)
			require.Zero(t, stepValueCount)

			manuscriptCount, err := db.NewSelect().
				Model((*dao.Manuscript)(nil)).
				Where("id = ?", manuscriptID).
				Count(ctx)
			require.NoError(t, err)
			require.Zero(t, manuscriptCount)
		},
	)
}
