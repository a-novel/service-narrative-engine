package core_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	ownerID      = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	projectID    = uuid.MustParse("00000000-0000-0000-0000-000000000201")
	generationID = uuid.MustParse("00000000-0000-0000-0000-000000000601")
	createdAt    = time.Date(2026, 7, 28, 12, 0, 0, 123456000, time.UTC)
	updatedAt    = createdAt.Add(time.Second)
	settledAt    = updatedAt.Add(time.Second)
	expiresAt    = settledAt.Add(30 * 24 * time.Hour)
)

var staticManuscriptValue = json.RawMessage(
	`{"blocks":[{"type":"text","metadata":{"source":"draft",` +
		`"plugin":{"name":"notes","version":1}},` +
		`"data":{"text":"The buried foghorn answers.",` +
		`"marks":[{"type":"italic","start":4,"end":10}]}}]}`,
)

func contentDocumentOfSize(size int) json.RawMessage {
	const (
		prefix = `{"payload":"`
		suffix = `"}`
	)

	return json.RawMessage(prefix + strings.Repeat("x", size-len(prefix)-len(suffix)) + suffix)
}

func projectFixture() *dao.Project {
	return &dao.Project{ID: projectID, OwnerID: ownerID, CreatedAt: createdAt}
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

	if status.Terminal() {
		generation.SettledAt = &settledAt
		generation.ExpiresAt = &expiresAt
	}

	return generation
}

func responsesOutput(t *testing.T, value any) []byte {
	t.Helper()

	text, err := json.Marshal(value)
	require.NoError(t, err)

	output, err := json.Marshal(map[string]any{
		"output": []any{map[string]any{
			"type": "message",
			"content": []any{map[string]any{
				"type": "output_text",
				"text": string(text),
			}},
		}},
	})
	require.NoError(t, err)

	return output
}

func responsesOutputText(t *testing.T, text string) []byte {
	t.Helper()

	output, err := json.Marshal(map[string]any{"output_text": text})
	require.NoError(t, err)

	return output
}

func responsesRefusal(t *testing.T) []byte {
	t.Helper()

	output, err := json.Marshal(map[string]any{
		"output": []any{map[string]any{
			"content": []any{map[string]any{
				"type":    "refusal",
				"refusal": "provider refusal",
			}},
		}},
	})
	require.NoError(t, err)

	return output
}
