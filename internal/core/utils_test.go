package core_test

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/lib"
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

func generationFixture(status core.GenerationStatus, output string) *lib.Generation {
	generation := &lib.Generation{
		ID:          generationID,
		OwnerID:     ownerID,
		Purpose:     core.GenerationPurposeStudio,
		Status:      string(status),
		Attempt:     1,
		MaxAttempts: 2,
		Output:      output,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	if status.Terminal() {
		generation.SettledAt = &settledAt
		generation.ExpiresAt = &expiresAt
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
