package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Project represents a user's project that contains multiple schema versions.
type Project struct {
	bun.BaseModel `bun:"table:projects"`

	// ID is the unique identifier for the project.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// Owner is the ID of the user who owns this project.
	Owner uuid.UUID `bun:"owner,type:uuid"`
	// Lang is the language of the project (ISO 639-1).
	Lang string `bun:"lang"`
	// Title is the title of the project.
	Title string `bun:"title"`
	// Workflow is a list of module strings that define the project's workflow.
	Workflow []string `bun:"workflow,array"`

	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}
