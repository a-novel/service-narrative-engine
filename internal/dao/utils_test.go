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

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/modelstest"
)

var (
	fixtureOwnerID         = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	fixtureProjectID       = uuid.MustParse("00000000-0000-0000-0000-000000000201")
	fixtureIdeaVersionID   = uuid.MustParse("00000000-0000-0000-0000-000000000202")
	fixtureEngineID        = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	fixtureEngineVersionID = uuid.MustParse("00000000-0000-0000-0000-000000000100")
	fixtureCreatedAt       = time.Date(2026, 7, 26, 0, 0, 0, 123456000, time.UTC)
)

var fixtureEngineDefinition = json.RawMessage(modelstest.WalkingSkeletonEngineDefinition)

func insertWalkingSkeletonFixtures(t *testing.T, ctx context.Context) {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	_, err = db.NewInsert().Model(&dao.Engine{
		ID:   fixtureEngineID,
		Kind: dao.EngineKindProject,
		Slug: "walking-skeleton",
	}).Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewInsert().Model(&dao.EngineVersion{
		ID:         fixtureEngineVersionID,
		EngineID:   fixtureEngineID,
		Version:    "0.0.1",
		Definition: fixtureEngineDefinition,
		CreatedAt:  fixtureCreatedAt,
	}).Exec(ctx)
	require.NoError(t, err)

	_, err = dao.NewPgIdeaInsert().Exec(ctx, &dao.IdeaInsertRequest{
		ProjectID: fixtureProjectID,
		VersionID: fixtureIdeaVersionID,
		OwnerID:   fixtureOwnerID,
		Seed:      "A lighthouse keeper hears a second foghorn answer from beneath the sea.",
		Genre:     "speculative",
		Title:     "The Answering Light",
		Now:       fixtureCreatedAt,
	})
	require.NoError(t, err)
}

func insertIdeaVersionHistoryFixtures(t *testing.T, ctx context.Context, count int) {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	versions := make([]*dao.IdeaVersion, 0, count)
	for index := range count {
		versions = append(versions, &dao.IdeaVersion{
			ID:        uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("idea-history-%02d", index))),
			ProjectID: fixtureProjectID,
			Title:     fmt.Sprintf("Title %02d", index),
			Genre:     "speculative",
			Seed:      fmt.Sprintf("Seed %02d", index),
			CreatedAt: fixtureCreatedAt.Add(time.Duration(index+1) * time.Second),
		})
	}

	_, err = db.NewInsert().Model(&versions).Exec(ctx)
	require.NoError(t, err)
}

func insertStepValueHistoryFixtures(
	t *testing.T,
	ctx context.Context,
	key string,
	count int,
) {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	values := make([]*dao.StepValue, 0, count)
	for index := range count {
		values = append(values, &dao.StepValue{
			ID:        uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("step-%s-%02d", key, index))),
			ProjectID: fixtureProjectID,
			Key:       key,
			Value:     json.RawMessage(fmt.Sprintf(`{"version":%d}`, index)),
			CreatedAt: fixtureCreatedAt.Add(time.Duration(index) * time.Second),
		})
	}

	_, err = db.NewInsert().Model(&values).Exec(ctx)
	require.NoError(t, err)
}

func insertManuscriptHistoryFixtures(t *testing.T, ctx context.Context, count int) {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	manuscripts := make([]*dao.Manuscript, 0, count)
	for index := range count {
		manuscripts = append(manuscripts, &dao.Manuscript{
			ID:        uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("manuscript-%02d", index))),
			ProjectID: fixtureProjectID,
			Value:     json.RawMessage(fmt.Sprintf(`{"blocks":[{"version":%d}]}`, index)),
			CreatedAt: fixtureCreatedAt.Add(time.Duration(index) * time.Second),
		})
	}

	_, err = db.NewInsert().Model(&manuscripts).Exec(ctx)
	require.NoError(t, err)
}
