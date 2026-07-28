package dao_test

import (
	"context"
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

	now := time.Date(2026, 7, 28, 1, 2, 3, 123456000, time.UTC)
	request := &dao.ManuscriptInsertRequest{
		ID:     uuid.MustParse("00000000-0000-0000-0000-000000000401"),
		IdeaID: fixtureIdeaID,
		Value:  fixtureManuscriptValue,
		Now:    now,
	}
	expect := &dao.Manuscript{
		ID:        request.ID,
		IdeaID:    request.IdeaID,
		CreatedAt: now,
	}

	postgres.RunIsolatedTransactionalTest(
		t,
		configtest.PostgresPreset,
		migrations.Migrations,
		func(ctx context.Context, t *testing.T) {
			t.Helper()

			insertWalkingSkeletonFixtures(t, ctx)

			manuscript, err := dao.NewPgManuscriptInsert().Exec(ctx, request)
			require.NoError(t, err)

			require.JSONEq(t, string(request.Value), string(manuscript.Value))
			manuscript.Value = nil
			require.Equal(t, expect, manuscript)
		},
	)
}
