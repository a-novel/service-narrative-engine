package core

import (
	"encoding/json"
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
	GenerationReasoningEffortDefault = "medium"

	generationJSONComponentMaxBytes   = 1 << 20
	generationProviderRequestMaxBytes = 1 << 20
	generationMaxAttempts             = 2
)

var (
	ErrGenerationConflict        = errors.New("generation idempotency conflict")
	ErrGenerationNotFound        = errors.New("generation not found")
	ErrGenerationOutputInvalid   = errors.New("invalid generation output")
	ErrGenerationRefused         = errors.New("generation refused")
	ErrGenerationResponseInvalid = errors.New("invalid generation response")
	ErrGenerationStatusUnknown   = errors.New("unknown generation status")
	ErrGenerationWatchClosed     = errors.New("generation watch closed before a terminal status")
)

// GenerationStatus is Narrative Engine's stable generation lifecycle vocabulary.
type GenerationStatus string

const (
	GenerationStatusPending   GenerationStatus = "pending"
	GenerationStatusRunning   GenerationStatus = "running"
	GenerationStatusSucceeded GenerationStatus = "succeeded"
	GenerationStatusFailed    GenerationStatus = "failed"
	GenerationStatusAbandoned GenerationStatus = "abandoned"
	GenerationStatusCancelled GenerationStatus = "cancelled"
)

var generationTerminalStatuses = map[GenerationStatus]struct{}{
	GenerationStatusSucceeded: {},
	GenerationStatusFailed:    {},
	GenerationStatusAbandoned: {},
	GenerationStatusCancelled: {},
}

// Terminal reports whether no further generation state can follow.
func (status GenerationStatus) Terminal() bool {
	_, terminal := generationTerminalStatuses[status]

	return terminal
}

// Generation is the owner-scoped lifecycle and opaque JSON proposal.
type Generation struct {
	ID          uuid.UUID        `json:"id"`
	Status      GenerationStatus `json:"status"`
	Attempt     int32            `json:"attempt"`
	MaxAttempts int32            `json:"maxAttempts"`
	Proposal    json.RawMessage  `json:"proposal"`
	Failure     string           `json:"failure"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
	SettledAt   *time.Time       `json:"settledAt"`
	ExpiresAt   *time.Time       `json:"expiresAt"`
}

// GenerationSubmitResult reports whether submission created work or replayed it.
type GenerationSubmitResult struct {
	Generation *Generation `json:"generation"`
	Created    bool        `json:"created"`
}
