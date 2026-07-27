package dao

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// EngineKind identifies the content contract an Engine Version implements.
type EngineKind string

const (
	// EngineKindProject requires output that converges on the Manuscript contract.
	EngineKindProject EngineKind = "project"
	// EngineKindCollection produces content reusable across Projects.
	EngineKindCollection EngineKind = "collection"
)

// EngineVersion is one immutable runtime definition.
type EngineVersion struct {
	bun.BaseModel `bun:"table:engine_versions,alias:engine_version"`

	// ID identifies the Engine Version.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// Kind identifies the content contract this version implements.
	Kind EngineKind `bun:"kind"`
	// Slug groups versions of the same Engine.
	Slug string `bun:"slug"`
	// Version is the semantic version of the Engine.
	Version string `bun:"version"`
	// Definition contains the runtime workflow definition.
	Definition json.RawMessage `bun:"definition,type:jsonb"`
	// CreatedAt records when the immutable version was published.
	CreatedAt time.Time `bun:"created_at"`
}
