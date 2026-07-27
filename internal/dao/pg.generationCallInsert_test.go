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

func TestPgGenerationCallInsert(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 26, 1, 2, 3, 123456000, time.UTC)
	call := &dao.GenerationCall{
		JobID:        uuid.MustParse("00000000-0000-0000-0000-000000000330"),
		Attempt:      1,
		OwnerID:      uuid.MustParse("00000000-0000-0000-0000-000000000042"),
		Provider:     "openai",
		Model:        "fixture-model",
		InputTokens:  10,
		OutputTokens: 20,
		CreatedAt:    now,
	}
	nextAttempt := *call
	nextAttempt.Attempt = 2
	nextAttempt.CreatedAt = now.Add(time.Second)

	insertCall := func(ctx context.Context, t *testing.T, generationCall *dao.GenerationCall) {
		t.Helper()

		db, err := postgres.GetContext(ctx)
		require.NoError(t, err)

		_, err = db.NewInsert().Model(generationCall).Exec(ctx)
		require.NoError(t, err)
	}

	testCases := []struct {
		name string

		request *dao.GenerationCallInsertRequest
		setup   func(context.Context, *testing.T)

		expect    *dao.GenerationCall
		expectErr error
	}{
		{
			name:    "Success",
			request: &dao.GenerationCallInsertRequest{Call: call},
			expect:  call,
		},
		{
			name:    "Success/NextAttempt",
			request: &dao.GenerationCallInsertRequest{Call: &nextAttempt},
			setup: func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertCall(ctx, t, call)
			},
			expect: &nextAttempt,
		},
		{
			name:    "Error/DuplicateAttempt",
			request: &dao.GenerationCallInsertRequest{Call: call},
			setup: func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertCall(ctx, t, call)
			},
			expectErr: dao.ErrGenerationCallInsertAttemptExists,
		},
	}

	operation := dao.NewPgGenerationCallInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunIsolatedTransactionalTest(
				t,
				configtest.PostgresPreset,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					if testCase.setup != nil {
						testCase.setup(ctx, t)
					}

					generationCall, err := operation.Exec(ctx, testCase.request)
					if testCase.expectErr != nil {
						require.ErrorIs(t, err, testCase.expectErr)
						require.Nil(t, generationCall)

						return
					}

					require.NoError(t, err)
					require.Equal(t, testCase.expect, generationCall)
				},
			)
		})
	}
}
