package core_test

import (
	_ "embed"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/modelstest"
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

//go:embed testdata/validation.engine.json
var validationEngineDefinition []byte

var engineDefinition = json.RawMessage(modelstest.WalkingSkeletonEngineDefinition)

var manuscriptValue = json.RawMessage(
	`{"title":"The Answering Light","format":"prose",` +
		`"scenes":[{"title":"The Reply","blocks":[` +
		`{"kind":"prose","text":"The buried foghorn answers."}]}]}`,
)

var staticManuscriptValue = json.RawMessage(
	`{"blocks":[{"type":"text","metadata":{"source":"draft",` +
		`"plugin":{"name":"notes","version":1}},` +
		`"data":{"text":"The buried foghorn answers.",` +
		`"marks":[{"type":"italic","start":4,"end":10}]}}]}`,
)

func ideaFixture() *dao.Idea {
	return &dao.Idea{
		ID:        ideaID,
		VersionID: ideaID,
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

func validationEngineVersionFixture() *dao.EngineVersion {
	return &dao.EngineVersion{
		ID:         engineVersionID,
		Definition: json.RawMessage(validationEngineDefinition),
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
	proposal json.RawMessage,
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

	if proposal != nil {
		target := generationTargetFixture()
		generation.Target = &target
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

	return generationEnvelopeForTarget(t, generationTargetFixture(), value)
}

func generationEnvelopeForTarget(
	t *testing.T,
	target core.GenerationTarget,
	value any,
) string {
	t.Helper()

	engineVersionID := ""
	stepKey := ""

	if target.Kind == core.GenerationTargetKindStep {
		engineVersionID = target.EngineVersionID.String()
		stepKey = target.StepKey
	}

	envelope, err := json.Marshal(map[string]any{
		"targetKind":      target.Kind,
		"engineVersionID": engineVersionID,
		"stepKey":         stepKey,
		"value":           value,
	})
	require.NoError(t, err)

	return string(envelope)
}

func generationTargetFixture() core.GenerationTarget {
	return core.GenerationTarget{
		Kind:            core.GenerationTargetKindStep,
		EngineVersionID: engineVersionID,
		StepKey:         "manuscript",
	}
}
