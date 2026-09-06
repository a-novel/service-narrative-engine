package dao_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
	"github.com/a-novel/service-narrative-engine/internal/models/modelstest"
)

func TestPgEngineVersionSelect(t *testing.T) {
	t.Parallel()

	engineVersionID := uuid.MustParse("00000000-0000-0000-0000-000000000100")
	engineID := uuid.MustParse("00000000-0000-0000-0000-000000000099")
	publishedAt := time.Date(2026, 7, 26, 1, 2, 3, 123456000, time.UTC)
	expectDefinition := string(modelstest.WalkingSkeletonEngineDefinition)
	engineFixture := &dao.Engine{
		ID:   engineID,
		Kind: dao.EngineKindProject,
		Slug: "walking-skeleton",
	}
	engineVersionFixture := &dao.EngineVersion{
		ID:         engineVersionID,
		EngineID:   engineID,
		Version:    "0.0.1",
		Definition: json.RawMessage(expectDefinition),
		CreatedAt:  publishedAt,
	}

	testCases := []struct {
		name string

		request *dao.EngineVersionSelectRequest

		expect    *dao.EngineVersion
		expectErr error
	}{
		{
			name:    "Success",
			request: &dao.EngineVersionSelectRequest{ID: engineVersionID},
			expect: &dao.EngineVersion{
				ID:        engineVersionID,
				EngineID:  engineID,
				Kind:      dao.EngineKindProject,
				Slug:      "walking-skeleton",
				Version:   "0.0.1",
				CreatedAt: publishedAt,
			},
		},
		{
			name: "Error/Absent",
			request: &dao.EngineVersionSelectRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-000000000399"),
			},
			expectErr: dao.ErrEngineVersionSelectNotFound,
		},
	}

	operation := dao.NewPgEngineVersionSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunIsolatedTransactionalTest(
				t,
				configtest.PostgresPreset,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					db, err := postgres.GetContext(ctx)
					require.NoError(t, err)

					_, err = db.NewInsert().Model(engineFixture).Exec(ctx)
					require.NoError(t, err)

					_, err = db.NewInsert().Model(engineVersionFixture).Exec(ctx)
					require.NoError(t, err)

					engineVersion, err := operation.Exec(ctx, testCase.request)
					require.ErrorIs(t, err, testCase.expectErr)

					// PostgreSQL re-serializes jsonb, so the definition is
					// compared semantically, then cleared so the remaining
					// columns settle in one comparison.
					if engineVersion != nil {
						require.JSONEq(t, expectDefinition, string(engineVersion.Definition))
						engineVersion.Definition = nil
					}

					require.Equal(t, testCase.expect, engineVersion)
				},
			)
		})
	}
}
