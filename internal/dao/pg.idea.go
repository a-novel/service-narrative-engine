package dao

import (
	"time"

	"github.com/google/uuid"
)

// Idea combines a stable project root with its latest typed content version.
type Idea struct {
	// ID identifies the stable Idea root.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// VersionID identifies the selected immutable content version.
	VersionID uuid.UUID `bun:"version_id,type:uuid"`
	// OwnerID identifies the user who owns the Idea.
	OwnerID uuid.UUID `bun:"owner_id,type:uuid"`
	// Seed is the source premise supplied by the writer.
	Seed string `bun:"seed"`
	// Genre selects the platform genre vocabulary.
	Genre string `bun:"genre"`
	// Title is the writer-supplied title, or an empty string when omitted.
	Title string `bun:"title"`
	// CreatedAt records when the Idea root was created.
	CreatedAt time.Time `bun:"created_at"`
	// UpdatedAt records when the selected content version was saved, or is nil for the initial version.
	UpdatedAt *time.Time `bun:"updated_at"`
}
