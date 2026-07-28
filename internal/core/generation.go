package core

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	// GenerationPurposeStudio is the closed usage dimension for Studio generation.
	GenerationPurposeStudio = "studio.generation"
	// GenerationModelDefault balances quality and cost for the Stage 2 fixture.
	GenerationModelDefault = "gpt-5.6-terra"
	// GenerationReasoningEffortDefault is the fixture's reasoning budget.
	GenerationReasoningEffortDefault = "low"
	generationMaxAttempts            = 2
)

var (
	ErrEngineVersionNotFound     = errors.New("engine version not found")
	ErrGenerationConflict        = errors.New("generation idempotency conflict")
	ErrGenerationNotFound        = errors.New("generation not found")
	ErrGenerationOutputInvalid   = errors.New("invalid generation output")
	ErrGenerationRefused         = errors.New("generation refused")
	ErrGenerationResponseInvalid = errors.New("invalid generation response")
	ErrGenerationStatusUnknown   = errors.New("unknown generation status")
	ErrGenerationWatchClosed     = errors.New("generation watch closed before a terminal status")

	errGenerationOutputEmpty       = errors.New("generation response output is empty")
	errGenerationOutputTextMissing = errors.New("generation response contains no output text")
	errGenerationOutputMultiple    = errors.New("generation response contains multiple JSON values")
	errProviderSchemaConflict      = errors.New("schema const is excluded by enum")
	errManuscriptInsertMissing     = errors.New("Manuscript insert returned no entity")
)

// GenerationStatus is narrative-engine's stable generation lifecycle vocabulary.
type GenerationStatus string

const (
	GenerationStatusPending   GenerationStatus = "pending"
	GenerationStatusRunning   GenerationStatus = "running"
	GenerationStatusSucceeded GenerationStatus = "succeeded"
	GenerationStatusFailed    GenerationStatus = "failed"
	GenerationStatusAbandoned GenerationStatus = "abandoned"
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
	case GenerationStatusPending, GenerationStatusRunning:
		return false
	default:
		return false
	}
}

// Generation is the owner-scoped lifecycle and volatile typed proposal.
type Generation struct {
	ID          uuid.UUID        `json:"id"`
	Status      GenerationStatus `json:"status"`
	Attempt     int32            `json:"attempt"`
	MaxAttempts int32            `json:"maxAttempts"`
	Proposal    *ManuscriptValue `json:"proposal"`
	Failure     string           `json:"failure"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
	SettledAt   *time.Time       `json:"settledAt"`
	ExpiresAt   *time.Time       `json:"expiresAt"`
}

// GenerationSubmitResult reports whether submission created work or replayed it.
type GenerationSubmitResult struct {
	Generation *Generation
	Created    bool
}
