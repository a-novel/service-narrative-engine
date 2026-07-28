package core_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	ownerID         = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	ideaID          = uuid.MustParse("00000000-0000-0000-0000-000000000201")
	engineVersionID = uuid.MustParse("00000000-0000-0000-0000-000000000100")
	generationID    = uuid.MustParse("00000000-0000-0000-0000-000000000601")
	createdAt       = time.Date(2026, 7, 28, 12, 0, 0, 123456000, time.UTC)
	updatedAt       = createdAt.Add(time.Second)
	settledAt       = updatedAt.Add(time.Second)
	expiresAt       = settledAt.Add(30 * 24 * time.Hour)
)

var engineDefinition = json.RawMessage(`{
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

var manuscriptValue = core.ManuscriptValue{
	Title:  "The Answering Light",
	Format: core.ManuscriptFormatProse,
	Scenes: []core.ManuscriptScene{{
		Title: "The Reply",
		Blocks: []core.ManuscriptBlock{{
			Kind: core.ManuscriptBlockKindProse,
			Text: "The buried foghorn answers.",
		}},
	}},
}

func ideaFixture() *dao.Idea {
	return &dao.Idea{
		ID:        ideaID,
		OwnerID:   ownerID,
		Seed:      "A lighthouse keeper hears a second foghorn answer from beneath the sea.",
		Genre:     "speculative",
		Title:     "The Answering Light",
		CreatedAt: createdAt,
	}
}

func engineVersionFixture() *dao.EngineVersion {
	return &dao.EngineVersion{
		ID:         engineVersionID,
		Definition: engineDefinition,
	}
}

func generationFixture(status servicegenai.GenerationStatus, output []byte) *servicegenai.Generation {
	generation := &servicegenai.Generation{
		Id:          generationID.String(),
		OwnerId:     ownerID.String(),
		Purpose:     core.GenerationPurposeStudio,
		Status:      status,
		Attempt:     1,
		MaxAttempts: 2,
		Output:      output,
		CreatedAt:   createdAt.Format(time.RFC3339Nano),
		UpdatedAt:   updatedAt.Format(time.RFC3339Nano),
	}

	switch status {
	case servicegenai.GenerationStatusSucceeded,
		servicegenai.GenerationStatusFailed,
		servicegenai.GenerationStatusAbandoned,
		servicegenai.GenerationStatusCancelled:
		generation.SettledAt = settledAt.Format(time.RFC3339Nano)
		generation.ExpiresAt = expiresAt.Format(time.RFC3339Nano)
	}

	return generation
}

func expectedGeneration(
	status core.GenerationStatus,
	proposal *core.ManuscriptValue,
) *core.Generation {
	generation := &core.Generation{
		ID:          generationID,
		Status:      status,
		Attempt:     1,
		MaxAttempts: 2,
		Proposal:    proposal,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	if status.Terminal() {
		generation.SettledAt = &settledAt
		generation.ExpiresAt = &expiresAt
	}

	return generation
}

func responsesOutput(t *testing.T, value any) []byte {
	t.Helper()

	output, err := json.Marshal(map[string]any{
		"output": []any{map[string]any{
			"type": "message",
			"content": []any{map[string]any{
				"type": "output_text",
				"text": generationEnvelope(t, value),
			}},
		}},
	})
	require.NoError(t, err)

	return output
}

func responsesTopLevelOutput(t *testing.T, value any) []byte {
	t.Helper()

	return responsesOutputText(t, generationEnvelope(t, value))
}

func responsesOutputText(t *testing.T, text string) []byte {
	t.Helper()

	output, err := json.Marshal(map[string]any{"output_text": text})
	require.NoError(t, err)

	return output
}

func generationEnvelope(t *testing.T, value any) string {
	t.Helper()

	envelope, err := json.Marshal(map[string]any{
		"engineVersionID": engineVersionID.String(),
		"stepKey":         "manuscript",
		"value":           value,
	})
	require.NoError(t, err)

	return string(envelope)
}
