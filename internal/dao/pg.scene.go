package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Scene is one column of a project's Scenes Composer: a unit of narrative order.
type Scene struct {
	bun.BaseModel `bun:"table:scenes,alias:scenes"`

	ID        uuid.UUID `bun:"id,pk,type:uuid"`
	ProjectID uuid.UUID `bun:"project_id,type:uuid"`
	// Position orders the project's scenes, from 0.
	Position int    `bun:"position"`
	Title    string `bun:"title"`
	Synopsis string `bun:"synopsis"`
	// Revision counts the writes to the scene's own fields. A move leaves it unchanged.
	Revision  int       `bun:"revision"`
	UpdatedAt time.Time `bun:"updated_at"`
}
