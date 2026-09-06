package dao_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
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

type concurrentContentKind string

const (
	concurrentContentIdea       concurrentContentKind = "idea"
	concurrentContentStep       concurrentContentKind = "step"
	concurrentContentManuscript concurrentContentKind = "manuscript"
)

// TestContentRetentionConcurrent proves the per-project lock keeps all three
// version histories at their exact cap when independent transactions race.
func TestContentRetentionConcurrent(t *testing.T) {
	t.Parallel()

	const (
		versionCount        = 32
		maxConcurrentWrites = 16
	)

	// Bound cross-case client use while preserving concurrent writes within each history.
	transactionSlots := make(chan struct{}, maxConcurrentWrites)

	testCases := []struct {
		name     string
		kind     concurrentContentKind
		idOffset int
	}{
		{name: "Idea", kind: concurrentContentIdea, idOffset: 2000},
		{name: "Step", kind: concurrentContentStep, idOffset: 3000},
		{name: "Manuscript", kind: concurrentContentManuscript, idOffset: 4000},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(
				t,
				configtest.PostgresPreset,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					insertWalkingSkeletonFixtures(t, ctx)

					start := make(chan struct{})
					errs := make([]error, versionCount)

					var (
						ready    sync.WaitGroup
						complete sync.WaitGroup
					)

					ready.Add(versionCount)
					complete.Add(versionCount)

					for index := range versionCount {
						go func() {
							defer complete.Done()

							ready.Done()
							<-start

							transactionSlots <- struct{}{}
							defer func() { <-transactionSlots }()

							id := concurrentContentVersionID(testCase.idOffset, index)
							errs[index] = postgres.WithinTx(ctx, nil, func(txCtx context.Context) error {
								return saveConcurrentContentVersion(txCtx, testCase.kind, index, id)
							})
						}()
					}

					ready.Wait()
					close(start)
					complete.Wait()

					for _, err := range errs {
						require.NoError(t, err)
					}

					ids, err := selectConcurrentContentVersionIDs(ctx, testCase.kind)
					require.NoError(t, err)
					require.Len(t, ids, 25)

					for index, id := range ids {
						expectedIndex := versionCount - 1 - index
						require.Equal(
							t,
							concurrentContentVersionID(testCase.idOffset, expectedIndex),
							id,
						)
					}
				},
			)
		})
	}
}

func concurrentContentVersionID(offset int, index int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf(
		"00000000-0000-0000-0000-%012d",
		offset+index,
	))
}

func saveConcurrentContentVersion(
	ctx context.Context,
	kind concurrentContentKind,
	index int,
	id uuid.UUID,
) error {
	now := fixtureCreatedAt.Add(time.Duration(index+1) * time.Second)

	switch kind {
	case concurrentContentIdea:
		_, err := dao.NewPgIdeaVersionInsert().Exec(ctx, &dao.IdeaVersionInsertRequest{
			ID:        id,
			ProjectID: fixtureProjectID,
			OwnerID:   fixtureOwnerID,
			Seed:      fmt.Sprintf("Revision %d", index),
			Genre:     "speculative",
			Title:     "The Answering Light",
			Now:       now,
		})

		return err
	case concurrentContentStep:
		_, err := dao.NewPgStepValueInsert().Exec(ctx, &dao.StepValueInsertRequest{
			ID:        id,
			ProjectID: fixtureProjectID,
			OwnerID:   fixtureOwnerID,
			Key:       "manuscript",
			Value:     json.RawMessage(fmt.Sprintf(`{"revision":%d}`, index)),
			Now:       now,
		})

		return err
	case concurrentContentManuscript:
		_, err := dao.NewPgManuscriptInsert().Exec(ctx, &dao.ManuscriptInsertRequest{
			ID:        id,
			ProjectID: fixtureProjectID,
			OwnerID:   fixtureOwnerID,
			Value:     json.RawMessage(fmt.Sprintf(`{"revision":%d}`, index)),
			Now:       now,
		})

		return err
	default:
		return fmt.Errorf("unknown concurrent content kind %q", kind)
	}
}

func selectConcurrentContentVersionIDs(
	ctx context.Context,
	kind concurrentContentKind,
) ([]uuid.UUID, error) {
	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, err
	}

	var ids []uuid.UUID

	switch kind {
	case concurrentContentIdea:
		err = db.NewSelect().
			Model((*dao.IdeaVersion)(nil)).
			Column("id").
			Where("project_id = ?", fixtureProjectID).
			OrderExpr("created_at DESC, id DESC").
			Scan(ctx, &ids)
	case concurrentContentStep:
		err = db.NewSelect().
			Model((*dao.StepValue)(nil)).
			Column("id").
			Where("project_id = ?", fixtureProjectID).
			Where("key = ?", "manuscript").
			OrderExpr("created_at DESC, id DESC").
			Scan(ctx, &ids)
	case concurrentContentManuscript:
		err = db.NewSelect().
			Model((*dao.Manuscript)(nil)).
			Column("id").
			Where("project_id = ?", fixtureProjectID).
			OrderExpr("created_at DESC, id DESC").
			Scan(ctx, &ids)
	default:
		return nil, fmt.Errorf("unknown concurrent content kind %q", kind)
	}

	if err != nil {
		return nil, fmt.Errorf("select retained %s versions: %w", kind, err)
	}

	return ids, nil
}
