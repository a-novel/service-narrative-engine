package dao

import (
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// EngineKind identifies the content contract an Engine implements.
type EngineKind string

const (
	// EngineKindProject requires output that converges on the Manuscript contract.
	EngineKindProject EngineKind = "project"
	// EngineKindCollection produces content reusable across Projects.
	EngineKindCollection EngineKind = "collection"
)

// Engine holds metadata shared by every immutable version of one Engine.
type Engine struct {
	bun.BaseModel `bun:"table:engines,alias:engine"`

	// ID identifies the Engine.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// Kind identifies the content contract implemented by every version.
	Kind EngineKind `bun:"kind"`
	// Slug is the stable human-readable Engine identifier.
	Slug string `bun:"slug"`
}
