package dao_test

import (
	"context"
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

func TestPgIdeaSelect(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 7, 26, 1, 2, 3, 0, time.UTC)
	updatedAt := createdAt.Add(time.Second)
	projectID := uuid.MustParse("00000000-0000-0000-0000-000000000311")
	initialVersionID := uuid.MustParse("00000000-0000-0000-0000-000000000312")
	latestVersionID := uuid.MustParse("00000000-0000-0000-0000-000000000313")
	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000042")
	idea := &dao.Idea{
		ProjectID:        projectID,
		VersionID:        latestVersionID,
		OwnerID:          ownerID,
		Seed:             "The answering foghorn moves closer.",
		Genre:            "speculative",
		Title:            "The Nearer Light",
		ProjectCreatedAt: createdAt,
		CreatedAt:        updatedAt,
	}

	testCases := []struct {
		name string

		request *dao.IdeaSelectRequest

		expect    *dao.Idea
		expectErr error
	}{
		{
			name: "Success/LatestVersion",
			request: &dao.IdeaSelectRequest{
				ProjectID: projectID,
				OwnerID:   ownerID,
			},
			expect: idea,
		},
		{
			name: "Error/OtherOwner",
			request: &dao.IdeaSelectRequest{
				ProjectID: projectID,
				OwnerID:   uuid.MustParse("00000000-0000-0000-0000-000000000043"),
			},
			expectErr: dao.ErrIdeaSelectNotFound,
		},
		{
			name: "Error/Absent",
			request: &dao.IdeaSelectRequest{
				ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000399"),
				OwnerID:   ownerID,
			},
			expectErr: dao.ErrIdeaSelectNotFound,
		},
	}

	operation := dao.NewPgIdeaSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunIsolatedTransactionalTest(
				t,
				configtest.PostgresPreset,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					_, err := dao.NewPgIdeaInsert().Exec(ctx, &dao.IdeaInsertRequest{
						ProjectID: projectID,
						VersionID: initialVersionID,
						OwnerID:   ownerID,
						Seed:      "A second foghorn answers from beneath the sea.",
						Genre:     idea.Genre,
						Title:     "The Answering Light",
						Now:       createdAt,
					})
					require.NoError(t, err)

					err = postgres.WithinTx(ctx, nil, func(ctx context.Context) error {
						_, err := dao.NewPgIdeaVersionInsert().Exec(ctx, &dao.IdeaVersionInsertRequest{
							ID:        latestVersionID,
							ProjectID: projectID,
							OwnerID:   ownerID,
							Seed:      idea.Seed,
							Genre:     idea.Genre,
							Title:     idea.Title,
							Now:       updatedAt,
						})

						return err
					})
					require.NoError(t, err)

					result, err := operation.Exec(ctx, testCase.request)
					require.ErrorIs(t, err, testCase.expectErr)
					require.Equal(t, testCase.expect, result)
				},
			)
		})
	}
}
