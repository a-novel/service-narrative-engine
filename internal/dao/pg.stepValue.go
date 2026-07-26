package dao

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// StepValue stores the accepted JSON value for one Engine step.
type StepValue struct {
	bun.BaseModel `bun:"table:step_values,alias:step_value"`

	// ID identifies the stored value.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// OwnerID identifies the user who owns the value.
	OwnerID uuid.UUID `bun:"owner_id,type:uuid"`
	// IdeaID identifies the Idea that produced the value.
	IdeaID uuid.UUID `bun:"idea_id,type:uuid"`
	// EngineVersionID identifies the definition that validated the value.
	EngineVersionID uuid.UUID `bun:"engine_version_id,type:uuid"`
	// StepKey selects the step within the Engine Version.
	StepKey string `bun:"step_key"`
	// GenerationJobID identifies the accepted generation.
	GenerationJobID uuid.UUID `bun:"generation_job_id,type:uuid"`
	// Value is the accepted step output.
	Value json.RawMessage `bun:"value,type:jsonb"`
	// CreatedAt records when the proposal was accepted.
	CreatedAt time.Time `bun:"created_at"`
}
