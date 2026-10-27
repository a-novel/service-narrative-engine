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
	// Color is the scene's color in the composer, as a lowercase #rrggbb hex code.
	Color string `bun:"color"`
	// Description says what the scene is for, written for the author.
	Description string `bun:"description"`
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt    string    `bun:"prompt"`
	UpdatedAt time.Time `bun:"updated_at"`
}
