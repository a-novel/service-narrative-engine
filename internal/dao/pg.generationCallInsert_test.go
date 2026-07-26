package dao_test

import (
	"context"
	"encoding/json"
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
	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000042")
	engineVersionID := uuid.MustParse("00000000-0000-0000-0000-000000000100")
	engineVersion := &dao.EngineVersion{
		ID:         engineVersionID,
		Slug:       "walking-skeleton",
		Version:    "0.0.1",
		Definition: json.RawMessage(`{"kind":"project","steps":[]}`),
		CreatedAt:  now,
	}
	idea := &dao.Idea{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000321"),
		OwnerID:   ownerID,
		Seed:      "A second foghorn answers from beneath the sea.",
		Genre:     "speculative",
		CreatedAt: now,
		UpdatedAt: now,
	}
	providerCallID := "resp_test"
	rawOutput := `{
  "title": "The Answering Light",
  "format": "prose",
  "scenes": [{
    "title": "The Reply",
    "blocks": [{"kind": "prose", "text": "The buried foghorn answers."}]
  }]
}`
	inputTokens := int64(10)
	outputTokens := int64(20)
	totalTokens := int64(30)

	call := &dao.GenerationCall{
		ID:                  uuid.MustParse("00000000-0000-0000-0000-000000000322"),
		JobID:               uuid.MustParse("00000000-0000-0000-0000-000000000330"),
		Attempt:             1,
		OwnerID:             ownerID,
		IdeaID:              idea.ID,
		EngineVersionID:     engineVersionID,
		Provider:            "openai",
		ProviderCallID:      &providerCallID,
		RequestHash:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Model:               "fixture-model",
		Outcome:             dao.GenerationOutcomeOK,
		RawOutput:           &rawOutput,
		InputTokens:         &inputTokens,
		OutputTokens:        &outputTokens,
		TotalTokens:         &totalTokens,
		LatencyMilliseconds: 250,
		CreatedAt:           now,
		CompletedAt:         now.Add(250 * time.Millisecond),
	}
	duplicateAttempt := *call
	duplicateAttempt.ID = uuid.MustParse("00000000-0000-0000-0000-000000000323")

	retryWithSameProviderCall := *call
	retryWithSameProviderCall.ID = uuid.MustParse("00000000-0000-0000-0000-000000000324")
	retryWithSameProviderCall.Attempt = 2

	missingIdea := *call
	missingIdea.ID = uuid.MustParse("00000000-0000-0000-0000-000000000325")
	missingIdea.JobID = uuid.MustParse("00000000-0000-0000-0000-000000000331")
	missingIdea.IdeaID = uuid.MustParse("00000000-0000-0000-0000-000000000399")

	missingEngineVersion := *call
	missingEngineVersion.ID = uuid.MustParse("00000000-0000-0000-0000-000000000326")
	missingEngineVersion.JobID = uuid.MustParse("00000000-0000-0000-0000-000000000332")
	missingEngineVersion.EngineVersionID = uuid.MustParse("00000000-0000-0000-0000-000000000398")

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
			name:    "Success/RetryReusesProviderCall",
			request: &dao.GenerationCallInsertRequest{Call: &retryWithSameProviderCall},
			setup: func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertCall(ctx, t, call)
			},
			expect: &retryWithSameProviderCall,
		},
		{
			name:    "Error/DuplicateAttempt",
			request: &dao.GenerationCallInsertRequest{Call: &duplicateAttempt},
			setup: func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertCall(ctx, t, call)
			},
			expectErr: dao.ErrGenerationCallInsertAlreadyExists,
		},
		{
			name:      "Error/IdeaNotFound",
			request:   &dao.GenerationCallInsertRequest{Call: &missingIdea},
			expectErr: dao.ErrGenerationCallInsertIdeaNotFound,
		},
		{
			name:      "Error/EngineVersionNotFound",
			request:   &dao.GenerationCallInsertRequest{Call: &missingEngineVersion},
			expectErr: dao.ErrGenerationCallInsertEngineVersionNotFound,
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

					_, err = db.NewInsert().Model(engineVersion).Exec(ctx)
					require.NoError(t, err)

					_, err = db.NewInsert().Model(idea).Exec(ctx)
					require.NoError(t, err)

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
