package dao

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Manuscript is one immutable client-saved Project document version.
type Manuscript struct {
	bun.BaseModel `bun:"table:manuscripts,alias:manuscript"`

	// ID identifies the Manuscript Version.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// ProjectID identifies the Project whose Manuscript was saved.
	ProjectID uuid.UUID `bun:"project_id,type:uuid"`
	// Value is the self-contained Manuscript document.
	Value json.RawMessage `bun:"value,type:jsonb"`
	// CreatedAt records when the version was saved.
	CreatedAt time.Time `bun:"created_at"`
}
