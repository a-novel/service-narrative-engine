package dao

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// StepValue is one immutable arbitrary JSON save under an opaque Project key.
type StepValue struct {
	bun.BaseModel `bun:"table:step_values,alias:step_value"`

	// ID identifies the saved version.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// ProjectID identifies the Project whose content was saved.
	ProjectID uuid.UUID `bun:"project_id,type:uuid"`
	// Key is an opaque client-controlled content identity.
	Key string `bun:"key"`
	// Value is arbitrary valid JSON.
	Value json.RawMessage `bun:"value,type:jsonb"`
	// CreatedAt records when the version was saved.
	CreatedAt time.Time `bun:"created_at"`
}
