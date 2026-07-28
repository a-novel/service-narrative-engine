package dao

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Manuscript is a self-contained, client-saved project document.
type Manuscript struct {
	bun.BaseModel `bun:"table:manuscripts,alias:manuscript"`

	// ID identifies the Manuscript.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// IdeaID identifies the Idea from which this project content was saved.
	IdeaID uuid.UUID `bun:"idea_id,type:uuid"`
	// Value is the opaque, self-contained Manuscript document.
	Value json.RawMessage `bun:"value,type:jsonb"`
	// CreatedAt records when the Manuscript was saved.
	CreatedAt time.Time `bun:"created_at"`
	// UpdatedAt records when the Manuscript last changed, or is nil before its first update.
	UpdatedAt *time.Time `bun:"updated_at"`
}
