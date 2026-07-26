package dao_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgEngineVersionSelect(t *testing.T) {
	t.Parallel()

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

	testCases := []struct {
		name string

		request *dao.EngineVersionSelectRequest

		expectSlug string
		expectErr  error
	}{
		{
			name:       "Success",
			request:    &dao.EngineVersionSelectRequest{ID: dao.FixtureEngineVersionID},
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

					engineVersion, err := operation.Exec(ctx, testCase.request)
					require.ErrorIs(t, err, testCase.expectErr)

					if testCase.expectErr != nil {
						require.Nil(t, engineVersion)

						return
					}

					require.Equal(t, dao.FixtureEngineVersionID, engineVersion.ID)
					require.Equal(t, testCase.expectSlug, engineVersion.Slug)
					require.Equal(t, "0.0.1", engineVersion.Version)
					require.Len(t, engineVersion.ContentHash, 64)
					require.JSONEq(t, expectDefinition, string(engineVersion.Definition))
				},
			)
		})
	}
}
