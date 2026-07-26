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

func TestPgEngineVersionSelect(t *testing.T) {
	t.Parallel()

	engineVersionID := uuid.MustParse("00000000-0000-0000-0000-000000000100")
	publishedAt := time.Date(2026, 7, 26, 1, 2, 3, 123456000, time.UTC)
	expectDefinition := `{
  "kind": "project",
  "steps": [{
    "key": "manuscript",
    "promptTemplate": "Turn the idea into a concise prose manuscript proposal.",
    "outputSchema": {
      "$schema": "https://json-schema.org/draft/2020-12/schema",
      "type": "object",
      "additionalProperties": false,
      "required": ["title", "format", "scenes"],
      "properties": {
        "title": {"type": "string", "minLength": 1},
        "format": {"const": "prose"},
        "scenes": {
          "type": "array",
          "minItems": 1,
          "items": {
            "type": "object",
            "additionalProperties": false,
            "required": ["title", "blocks"],
            "properties": {
              "title": {"type": "string", "minLength": 1},
              "blocks": {
                "type": "array",
                "minItems": 1,
                "items": {
                  "type": "object",
                  "additionalProperties": false,
                  "required": ["kind", "text"],
                  "properties": {
                    "kind": {"enum": ["prose", "dialogue", "cue"]},
                    "text": {"type": "string", "minLength": 1}
                  }
                }
              }
            }
          }
        }
      }
    }
  }]
}`
	engineVersionFixture := &dao.EngineVersion{
		ID:         engineVersionID,
		Slug:       "walking-skeleton",
		Version:    "0.0.1",
		Definition: json.RawMessage(expectDefinition),
		CreatedAt:  publishedAt,
	}

	testCases := []struct {
		name string

		request *dao.EngineVersionSelectRequest

		expectSlug string
		expectErr  error
	}{
		{
			name:       "Success",
			request:    &dao.EngineVersionSelectRequest{ID: engineVersionID},
			expectSlug: "walking-skeleton",
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

			postgres.RunIsolatedTransactionalTest(
				t,
				configtest.PostgresPreset,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					db, err := postgres.GetContext(ctx)
					require.NoError(t, err)

					_, err = db.NewInsert().Model(engineVersionFixture).Exec(ctx)
					require.NoError(t, err)

					engineVersion, err := operation.Exec(ctx, testCase.request)
					require.ErrorIs(t, err, testCase.expectErr)

					if testCase.expectErr != nil {
						require.Nil(t, engineVersion)

						return
					}

					require.Equal(t, engineVersionID, engineVersion.ID)
					require.Equal(t, testCase.expectSlug, engineVersion.Slug)
					require.Equal(t, "0.0.1", engineVersion.Version)
					require.Equal(t, publishedAt, engineVersion.CreatedAt)
					require.JSONEq(t, expectDefinition, string(engineVersion.Definition))
				},
			)
		})
	}
}
