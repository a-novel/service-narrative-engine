package dao

import (
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ManuscriptBlock is one ordered prose, dialogue, or cue block in a scene.
type ManuscriptBlock struct {
	bun.BaseModel `bun:"table:manuscript_blocks,alias:manuscript_block"`

	// ID identifies the block.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// SceneID identifies the containing scene.
	SceneID uuid.UUID `bun:"scene_id,type:uuid"`
	// Ordinal is the block zero-based position.
	Ordinal int `bun:"ordinal"`
	// Kind selects the renderer behavior for the block.
	Kind string `bun:"kind"`
	// Text is the block authored content.
	Text string `bun:"text"`
}
