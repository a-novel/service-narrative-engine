package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// IdeaVersion is one immutable typed-content save under a Project.
type IdeaVersion struct {
	bun.BaseModel `bun:"table:idea_versions,alias:idea_version"`

	// ID identifies the content version.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// ProjectID identifies the stable Project.
	ProjectID uuid.UUID `bun:"project_id,type:uuid"`
	// Seed is the source premise supplied by the writer.
	Seed string `bun:"seed"`
	// Genre selects the platform genre vocabulary.
	Genre string `bun:"genre"`
	// Title is the writer-supplied title, or an empty string when omitted.
	Title string `bun:"title"`
	// CreatedAt records when the content version was saved.
	CreatedAt time.Time `bun:"created_at"`
}
