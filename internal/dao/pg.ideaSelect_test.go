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

func TestPgIdeaSelect(t *testing.T) {
	t.Parallel()

	idea := &dao.Idea{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000311"),
		OwnerID:   uuid.MustParse("00000000-0000-0000-0000-000000000042"),
		Seed:      "A second foghorn answers from beneath the sea.",
		StoryType: "novel",
		Genre:     "speculative",
		CreatedAt: time.Date(2026, 7, 26, 1, 2, 3, 0, time.UTC),
		UpdatedAt: time.Date(2026, 7, 26, 1, 2, 3, 0, time.UTC),
	}

	testCases := []struct {
		name string

		request *dao.IdeaSelectRequest

		expect    *dao.Idea
		expectErr error
	}{
		{
			name: "Success",
			request: &dao.IdeaSelectRequest{
				ID:      idea.ID,
				OwnerID: idea.OwnerID,
			},
			expect: idea,
		},
		{
			name: "Error/OtherOwner",
			request: &dao.IdeaSelectRequest{
				ID:      idea.ID,
				OwnerID: uuid.MustParse("00000000-0000-0000-0000-000000000043"),
			},
			expectErr: dao.ErrIdeaSelectNotFound,
		},
		{
			name: "Error/Absent",
			request: &dao.IdeaSelectRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000399"),
				OwnerID: idea.OwnerID,
			},
			expectErr: dao.ErrIdeaSelectNotFound,
		},
	}

	operation := dao.NewPgIdeaSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunIsolatedTransactionalTest(
				t,
				configtest.PostgresPreset,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					db, err := postgres.GetContext(ctx)
					require.NoError(t, err)

					_, err = db.NewInsert().Model(idea).Exec(ctx)
					require.NoError(t, err)

					result, err := operation.Exec(ctx, testCase.request)
					require.ErrorIs(t, err, testCase.expectErr)
					require.Equal(t, testCase.expect, result)
				},
			)
		})
	}
}
