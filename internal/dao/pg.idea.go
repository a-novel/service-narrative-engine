package dao

import (
	"time"

	"github.com/google/uuid"
)

// Idea combines one immutable content version with its stable Project identity.
type Idea struct {
	// ProjectID identifies the stable Project.
	ProjectID uuid.UUID `bun:"project_id,type:uuid"`
	// VersionID identifies the selected Idea Version.
	VersionID uuid.UUID `bun:"version_id,pk,type:uuid"`
	// OwnerID identifies the user who owns the Project.
	OwnerID uuid.UUID `bun:"owner_id,type:uuid"`
	// Seed is the source premise supplied by the writer.
	Seed string `bun:"seed"`
	// Genre selects the platform genre vocabulary.
	Genre string `bun:"genre"`
	// Title is the writer-supplied title, or an empty string when omitted.
	Title string `bun:"title"`
	// ProjectCreatedAt records when the stable Project was created.
	ProjectCreatedAt time.Time `bun:"project_created_at"`
	// CreatedAt records when this Idea Version was saved.
	CreatedAt time.Time `bun:"created_at"`
}
