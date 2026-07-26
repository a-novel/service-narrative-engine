package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Idea is the typed entry contract from which an Engine Version generates content.
type Idea struct {
	bun.BaseModel `bun:"table:ideas,alias:idea"`

	// ID identifies the Idea.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// OwnerID identifies the user who owns the Idea.
	OwnerID uuid.UUID `bun:"owner_id,type:uuid"`
	// Seed is the source premise supplied by the writer.
	Seed string `bun:"seed"`
	// StoryType selects the platform story shape.
	StoryType string `bun:"story_type"`
	// Genre selects the platform genre vocabulary.
	Genre string `bun:"genre"`
	// Title is the optional title supplied by the writer.
	Title *string `bun:"title"`
	// CreatedAt records when the Idea was created.
	CreatedAt time.Time `bun:"created_at"`
	// UpdatedAt records when the Idea last changed.
	UpdatedAt time.Time `bun:"updated_at"`
}
