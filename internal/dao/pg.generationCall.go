package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// GenerationCall records the pricing inputs reported for one provider attempt.
type GenerationCall struct {
	bun.BaseModel `bun:"table:generation_calls,alias:generation_call"`

	// JobID identifies the owning service-jobs record.
	JobID uuid.UUID `bun:"job_id,pk,type:uuid"`
	// Attempt is the service-jobs Job.Attempt value received for this execution.
	Attempt int `bun:"attempt,pk"`
	// OwnerID identifies the user billed for the generation.
	OwnerID uuid.UUID `bun:"owner_id,type:uuid"`
	// Provider names the model provider.
	Provider string `bun:"provider"`
	// Model is the actual model identifier returned by the provider.
	Model string `bun:"model"`
	// InputTokens records billed input usage.
	InputTokens int64 `bun:"input_tokens"`
	// OutputTokens records billed output usage.
	OutputTokens int64 `bun:"output_tokens"`
	// CreatedAt records when the billed attempt occurred.
	CreatedAt time.Time `bun:"created_at"`
}
