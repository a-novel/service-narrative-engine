package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// GenerationCall records the provider exchange that spent model capacity for one job.
type GenerationCall struct {
	bun.BaseModel `bun:"table:generation_calls,alias:generation_call"`

	// JobID is both the generation identifier and the owning service-jobs record.
	JobID uuid.UUID `bun:"job_id,pk,type:uuid"`
	// OwnerID identifies the user billed for the generation.
	OwnerID uuid.UUID `bun:"owner_id,type:uuid"`
	// IdeaID identifies the input Idea.
	IdeaID uuid.UUID `bun:"idea_id,type:uuid"`
	// EngineVersionID identifies the definition used to build the request.
	EngineVersionID uuid.UUID `bun:"engine_version_id,type:uuid"`
	// Provider names the model provider.
	Provider string `bun:"provider"`
	// ProviderCallID identifies the resumable provider operation when one exists.
	ProviderCallID *string `bun:"provider_call_id"`
	// RequestHash is the lowercase hexadecimal SHA-256 digest of the stable request.
	RequestHash string `bun:"request_hash"`
	// Model is the actual model identifier returned by the provider.
	Model string `bun:"model"`
	// Outcome classifies the terminal provider response.
	Outcome string `bun:"outcome"`
	// RawOutput preserves the provider payload for audit and replay.
	RawOutput *string `bun:"raw_output"`
	// InputTokens records billed input usage when the provider reports it.
	InputTokens *int64 `bun:"input_tokens"`
	// OutputTokens records billed output usage when the provider reports it.
	OutputTokens *int64 `bun:"output_tokens"`
	// TotalTokens records aggregate billed usage when the provider reports it.
	TotalTokens *int64 `bun:"total_tokens"`
	// LatencyMilliseconds records the elapsed provider-operation duration.
	LatencyMilliseconds int64 `bun:"latency_ms"`
	// Refusal preserves refusal detail returned by the provider.
	Refusal *string `bun:"refusal"`
	// Error preserves terminal error detail returned by the provider.
	Error *string `bun:"error"`
	// CreatedAt records when the provider operation began.
	CreatedAt time.Time `bun:"created_at"`
	// CompletedAt records when the provider operation reached its terminal outcome.
	CompletedAt time.Time `bun:"completed_at"`
}
