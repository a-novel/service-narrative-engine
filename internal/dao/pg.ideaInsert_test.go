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

func TestPgIdeaInsert(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 26, 1, 2, 3, 0, time.UTC)
	testCases := []struct {
		name string

		request *dao.IdeaInsertRequest

		expect *dao.Idea
	}{
		{
			name: "Success",

			request: &dao.IdeaInsertRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000301"),
				OwnerID: uuid.MustParse("00000000-0000-0000-0000-000000000042"),
				Seed:    "A second foghorn answers from beneath the sea.",
				Genre:   "speculative",
				Title:   "The Answering Light",
				Now:     now,
			},

			expect: &dao.Idea{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000301"),
				OwnerID:   uuid.MustParse("00000000-0000-0000-0000-000000000042"),
				Seed:      "A second foghorn answers from beneath the sea.",
				Genre:     "speculative",
				Title:     "The Answering Light",
				CreatedAt: now,
			},
		},
		{
			name: "Success/WithoutTitle",

			request: &dao.IdeaInsertRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000302"),
				OwnerID: uuid.MustParse("00000000-0000-0000-0000-000000000042"),
				Seed:    "A city wakes with no shadows.",
				Genre:   "speculative",
				Now:     now,
			},

			expect: &dao.Idea{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000302"),
				OwnerID:   uuid.MustParse("00000000-0000-0000-0000-000000000042"),
				Seed:      "A city wakes with no shadows.",
				Genre:     "speculative",
				CreatedAt: now,
			},
		},
	}

	operation := dao.NewPgIdeaInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunIsolatedTransactionalTest(
				t,
				configtest.PostgresPreset,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					idea, err := operation.Exec(ctx, testCase.request)
					require.NoError(t, err)
					require.Equal(t, testCase.expect, idea)
				},
			)
		})
	}
}
