package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Layer is one row of a project's Scenes Composer: one kind of information followed across scenes.
type Layer struct {
	bun.BaseModel `bun:"table:layers,alias:layers"`

	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// ProjectID is the project the layer belongs to.
	ProjectID uuid.UUID `bun:"project_id,type:uuid"`

	// Position orders the project's layers, from 0.
	Position int `bun:"position"`
	// Kind names the content schema the layer's cells follow.
	Kind string `bun:"kind"`

	// Name labels the layer's row in the composer.
	Name string `bun:"name"`
	// Description says what the layer records.
	Description string `bun:"description"`

	UpdatedAt time.Time `bun:"updated_at"`
}
