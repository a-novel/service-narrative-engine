package dao

import (
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ManuscriptScene is one ordered scene in a Manuscript.
type ManuscriptScene struct {
	bun.BaseModel `bun:"table:manuscript_scenes,alias:manuscript_scene"`

	// ID identifies the scene.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// ManuscriptID identifies the containing Manuscript.
	ManuscriptID uuid.UUID `bun:"manuscript_id,type:uuid"`
	// Ordinal is the scene zero-based position.
	Ordinal int `bun:"ordinal"`
	// Title is the scene display title.
	Title string `bun:"title"`
}
