package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Scene is one subcolumn of a project's Scenes Composer: a part of a beat.
type Scene struct {
	bun.BaseModel `bun:"table:scenes,alias:scenes"`

	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// ProjectID is the project the scene belongs to.
	ProjectID uuid.UUID `bun:"project_id,type:uuid"`
	// BeatID is the beat the scene is part of.
	BeatID uuid.UUID `bun:"beat_id,type:uuid"`

	// Position orders the beat's scenes, from 0.
	Position int `bun:"position"`

	// Title names the scene in the composer.
	Title string `bun:"title"`
	// Description refines the beat's description for the scene. Empty when the beat's is enough.
	Description string `bun:"description"`
	// Prompt refines the beat's prompt for the scene. Empty when the beat's is enough.
	Prompt string `bun:"prompt"`

	UpdatedAt time.Time `bun:"updated_at"`
}
