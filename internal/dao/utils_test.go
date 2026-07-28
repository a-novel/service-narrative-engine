package dao_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	fixtureOwnerID         = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	fixtureIdeaID          = uuid.MustParse("00000000-0000-0000-0000-000000000201")
	fixtureEngineID        = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	fixtureEngineVersionID = uuid.MustParse("00000000-0000-0000-0000-000000000100")
	fixtureCreatedAt       = time.Date(2026, 7, 26, 0, 0, 0, 123456000, time.UTC)
)

var fixtureEngineDefinition = json.RawMessage(`{
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
}`)

var fixtureManuscriptValue = json.RawMessage(`{
  "title": "The Answering Light",
  "format": "prose",
  "scenes": [{
    "title": "The Reply",
    "blocks": [{
      "kind": "prose",
      "text": "The buried foghorn answers."
    }]
  }]
}`)

func insertWalkingSkeletonFixtures(t *testing.T, ctx context.Context) {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	_, err = db.NewInsert().Model(&dao.Engine{
		ID:   fixtureEngineID,
		Kind: dao.EngineKindProject,
		Slug: "walking-skeleton",
	}).Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewInsert().Model(&dao.EngineVersion{
		ID:         fixtureEngineVersionID,
		EngineID:   fixtureEngineID,
		Version:    "0.0.1",
		Definition: fixtureEngineDefinition,
		CreatedAt:  fixtureCreatedAt,
	}).Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewInsert().Model(&dao.Idea{
		ID:        fixtureIdeaID,
		OwnerID:   fixtureOwnerID,
		Seed:      "A lighthouse keeper hears a second foghorn answer from beneath the sea.",
		Genre:     "speculative",
		Title:     "The Answering Light",
		CreatedAt: fixtureCreatedAt,
	}).Exec(ctx)
	require.NoError(t, err)
}
