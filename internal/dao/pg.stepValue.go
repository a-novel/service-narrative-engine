package dao

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// StepValue is client-saved content for one immutable Engine Version step.
type StepValue struct {
	bun.BaseModel `bun:"table:step_values,alias:step_value"`

	// ID identifies the saved value.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// IdeaID identifies the Idea whose content was saved.
	IdeaID uuid.UUID `bun:"idea_id,type:uuid"`
	// EngineVersionID identifies the immutable definition that owns the step.
	EngineVersionID uuid.UUID `bun:"engine_version_id,type:uuid"`
	// StepKey identifies the step inside the Engine Version definition.
	StepKey string `bun:"step_key"`
	// Value is the schema-validated, source-agnostic content.
	Value json.RawMessage `bun:"value,type:jsonb"`
	// CreatedAt records when the value was saved.
	CreatedAt time.Time `bun:"created_at"`
}
