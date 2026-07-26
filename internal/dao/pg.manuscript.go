package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Manuscript is the typed exit contract produced by an accepted generation.
type Manuscript struct {
	bun.BaseModel `bun:"table:manuscripts,alias:manuscript"`

	// ID identifies the Manuscript.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// OwnerID identifies the user who owns the Manuscript.
	OwnerID uuid.UUID `bun:"owner_id,type:uuid"`
	// IdeaID identifies the entry contract that produced the Manuscript.
	IdeaID uuid.UUID `bun:"idea_id,type:uuid"`
	// AcceptedGenerationJobID identifies the proposal accepted into this Manuscript.
	AcceptedGenerationJobID uuid.UUID `bun:"accepted_generation_job_id,type:uuid"`
	// Title is the Manuscript display title.
	Title string `bun:"title"`
	// Format constrains the block kinds accepted by the renderer.
	Format string `bun:"format"`
	// CreatedAt records when the Manuscript was accepted.
	CreatedAt time.Time `bun:"created_at"`
	// UpdatedAt records when the Manuscript last changed.
	UpdatedAt time.Time `bun:"updated_at"`
}
