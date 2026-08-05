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
	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000042")
	testCases := []struct {
		name string

		request *dao.IdeaInsertRequest

		expect *dao.Idea
	}{
		{
			name: "Success",
			request: &dao.IdeaInsertRequest{
				ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000301"),
				VersionID: uuid.MustParse("00000000-0000-0000-0000-000000000311"),
				OwnerID:   ownerID,
				Seed:      "A second foghorn answers from beneath the sea.",
				Genre:     "speculative",
				Title:     "The Answering Light",
				Now:       now,
			},
			expect: &dao.Idea{
				ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000301"),
				VersionID:        uuid.MustParse("00000000-0000-0000-0000-000000000311"),
				OwnerID:          ownerID,
				Seed:             "A second foghorn answers from beneath the sea.",
				Genre:            "speculative",
				Title:            "The Answering Light",
				ProjectCreatedAt: now,
				CreatedAt:        now,
			},
		},
		{
			name: "Success/WithoutTitle",
			request: &dao.IdeaInsertRequest{
				ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000302"),
				VersionID: uuid.MustParse("00000000-0000-0000-0000-000000000312"),
				OwnerID:   ownerID,
				Seed:      "A city wakes with no shadows.",
				Genre:     "speculative",
				Now:       now,
			},
			expect: &dao.Idea{
				ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000302"),
				VersionID:        uuid.MustParse("00000000-0000-0000-0000-000000000312"),
				OwnerID:          ownerID,
				Seed:             "A city wakes with no shadows.",
				Genre:            "speculative",
				ProjectCreatedAt: now,
				CreatedAt:        now,
			},
		},
		{
			name: "Success/EmptyPartialIdea",
			request: &dao.IdeaInsertRequest{
				ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000303"),
				VersionID: uuid.MustParse("00000000-0000-0000-0000-000000000313"),
				OwnerID:   ownerID,
				Now:       now,
			},
			expect: &dao.Idea{
				ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000303"),
				VersionID:        uuid.MustParse("00000000-0000-0000-0000-000000000313"),
				OwnerID:          ownerID,
				ProjectCreatedAt: now,
				CreatedAt:        now,
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
