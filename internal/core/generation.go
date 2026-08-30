package core

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	// GenerationPurposeStudio identifies Studio content generation in service-genai.
	GenerationPurposeStudio = "studio.generation"
	// GenerationModelDefault selects the model used for Studio content generation.
	GenerationModelDefault = "gpt-5.6-terra"
	// GenerationReasoningEffortDefault selects the reasoning budget for Studio generation.
	GenerationReasoningEffortDefault = "medium"
	// GenerationMaxOutputTokensDefault bounds output spend while leaving room for long story fragments.
	GenerationMaxOutputTokensDefault int64 = 32_768

	generationJSONComponentMaxBytes   = 1 << 20
	generationProviderRequestMaxBytes = 1 << 20
	generationMaxAttempts             = 2
)

var (
	// ErrGenerationConflict is returned when an idempotency key already identifies different work.
	ErrGenerationConflict = errors.New("generation idempotency conflict")
	// ErrGenerationNotFound is returned when service-genai has no matching owner-scoped generation.
	ErrGenerationNotFound = errors.New("generation not found")
	// ErrGenerationOutputInvalid is returned when successful provider output is not bounded JSON.
	ErrGenerationOutputInvalid = errors.New("invalid generation output")
	// ErrGenerationRefused is returned when the provider refuses a generation.
	ErrGenerationRefused = errors.New("generation refused")
	// ErrGenerationResponseInvalid is returned when service-genai breaks its response contract.
	ErrGenerationResponseInvalid = errors.New("invalid generation response")
	// ErrGenerationStatusUnknown is returned for a service-genai status this service cannot expose.
	ErrGenerationStatusUnknown = errors.New("unknown generation status")
	// ErrGenerationWatchClosed is returned when a stream ends before generation settles.
	ErrGenerationWatchClosed = errors.New("generation watch closed before a terminal status")
)

// GenerationStatus is Narrative Engine's stable generation lifecycle vocabulary.
type GenerationStatus string

const (
	// GenerationStatusPending means the generation is waiting for a worker.
	GenerationStatusPending GenerationStatus = "pending"
	// GenerationStatusRunning means a worker is executing the generation.
	GenerationStatusRunning GenerationStatus = "running"
	// GenerationStatusSucceeded means the generation produced a proposal.
	GenerationStatusSucceeded GenerationStatus = "succeeded"
	// GenerationStatusFailed means every provider attempt failed.
	GenerationStatusFailed GenerationStatus = "failed"
	// GenerationStatusAbandoned means the generation can no longer be executed.
	GenerationStatusAbandoned GenerationStatus = "abandoned"
	// GenerationStatusCancelled means the generation was cancelled before completion.
	GenerationStatusCancelled GenerationStatus = "cancelled"
)

// Terminal reports whether no further generation state can follow.
func (status GenerationStatus) Terminal() bool {
	switch status {
	case GenerationStatusSucceeded,
		GenerationStatusFailed,
		GenerationStatusAbandoned,
		GenerationStatusCancelled:
		return true
	default:
		return false
	}
}

// Generation is the owner-scoped lifecycle and opaque JSON proposal.
type Generation struct {
	// ID identifies the generation in service-genai.
	ID uuid.UUID `json:"id"`
	// Status is the current lifecycle state.
	Status GenerationStatus `json:"status"`
	// Attempt is the number of provider runs already started.
	Attempt int32 `json:"attempt"`
	// MaxAttempts is the maximum number of provider runs allowed.
	MaxAttempts int32 `json:"maxAttempts"`
	// Proposal is the opaque JSON produced by a successful generation.
	Proposal json.RawMessage `json:"proposal"`
	// Failure is an opaque terminal failure safe to return to the client.
	Failure string `json:"failure"`
	// CreatedAt records when service-genai accepted the generation.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt records the latest lifecycle change.
	UpdatedAt time.Time `json:"updatedAt"`
	// SettledAt records when the generation reached a terminal state.
	SettledAt *time.Time `json:"settledAt"`
	// ExpiresAt records when service-genai may purge the generation.
	ExpiresAt *time.Time `json:"expiresAt"`
}

// GenerationSubmitResult reports whether submission created work or replayed it.
type GenerationSubmitResult struct {
	// Generation is the created or replayed owner-scoped work.
	Generation *Generation `json:"generation"`
	// Created distinguishes new work from an idempotent replay.
	Created bool `json:"created"`
}
