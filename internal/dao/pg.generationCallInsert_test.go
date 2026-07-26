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

	now := time.Date(2026, 7, 26, 1, 2, 3, 0, time.UTC)
	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000042")
	idea := &dao.Idea{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000321"),
		OwnerID:   ownerID,
		Seed:      "A second foghorn answers from beneath the sea.",
		Genre:     "speculative",
		CreatedAt: now,
		UpdatedAt: now,
	}
	providerCallID := "resp_test"
	rawOutput := `{"title":"The Answering Light","format":"prose","scenes":[]}`
	inputTokens := int64(10)
	outputTokens := int64(20)
	totalTokens := int64(30)

	call := &dao.GenerationCall{
		JobID:               uuid.MustParse("00000000-0000-0000-0000-000000000322"),
		OwnerID:             ownerID,
		IdeaID:              idea.ID,
		EngineVersionID:     dao.FixtureEngineVersionID,
		Provider:            "openai",
		ProviderCallID:      &providerCallID,
		RequestHash:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Model:               "fixture-model",
		Outcome:             "ok",
		RawOutput:           &rawOutput,
		InputTokens:         &inputTokens,
		OutputTokens:        &outputTokens,
		TotalTokens:         &totalTokens,
		LatencyMilliseconds: 250,
		CreatedAt:           now,
		CompletedAt:         now.Add(time.Second),
	}
	duplicateProviderCall := *call
	duplicateProviderCall.JobID = uuid.MustParse("00000000-0000-0000-0000-000000000324")

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
		expectErr bool
	}{
		{
			name:    "Success",
			request: &dao.GenerationCallInsertRequest{Call: call},
			expect:  call,
		},
		{
			name:    "Error/DuplicateJob",
			request: &dao.GenerationCallInsertRequest{Call: call},
			setup: func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertCall(ctx, t, call)
			},
			expectErr: true,
		},
		{
			name:    "Error/DuplicateProviderCall",
			request: &dao.GenerationCallInsertRequest{Call: &duplicateProviderCall},
			setup: func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertCall(ctx, t, call)
			},
			expectErr: true,
		},
		{
			name: "Error/CrossOwnerIdea",
			request: &dao.GenerationCallInsertRequest{Call: &dao.GenerationCall{
				JobID:               uuid.MustParse("00000000-0000-0000-0000-000000000323"),
				OwnerID:             uuid.MustParse("00000000-0000-0000-0000-000000000043"),
				IdeaID:              idea.ID,
				EngineVersionID:     dao.FixtureEngineVersionID,
				Provider:            "openai",
				RequestHash:         "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Model:               "fixture-model",
				Outcome:             "error",
				LatencyMilliseconds: 250,
				CreatedAt:           now,
				CompletedAt:         now.Add(time.Second),
			}},
			expectErr: true,
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

					db, err := postgres.GetContext(ctx)
					require.NoError(t, err)

					_, err = db.NewInsert().Model(idea).Exec(ctx)
					require.NoError(t, err)

					if testCase.setup != nil {
						testCase.setup(ctx, t)
					}

					generationCall, err := operation.Exec(ctx, testCase.request)
					if testCase.expectErr {
						require.Error(t, err)
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
