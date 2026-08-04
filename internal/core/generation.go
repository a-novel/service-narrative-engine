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

	errProviderSchemaConflict    = errors.New("schema const is excluded by enum")
	errProviderSchemaUnsupported = errors.New("provider schema is unsupported")
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

var generationTerminalStatuses = map[GenerationStatus]struct{}{
	GenerationStatusSucceeded: {},
	GenerationStatusFailed:    {},
	GenerationStatusAbandoned: {},
	GenerationStatusCancelled: {},
}

// GenerationTargetKind identifies one of the three project content contracts.
type GenerationTargetKind string

const (
	// GenerationTargetKindIdea uses the static Idea schema.
	GenerationTargetKindIdea GenerationTargetKind = "idea"
	// GenerationTargetKindStep uses one immutable Engine Version step schema.
	GenerationTargetKindStep GenerationTargetKind = "step"
	// GenerationTargetKindManuscript uses the static Manuscript schema.
	GenerationTargetKindManuscript GenerationTargetKind = "manuscript"
)

// GenerationTarget identifies the content contract a proposal must complete.
type GenerationTarget struct {
	Kind            GenerationTargetKind `json:"kind"            validate:"required,oneof=idea step manuscript"`
	EngineVersionID uuid.UUID            `json:"engineVersionID"`
	StepKey         string               `json:"stepKey"         validate:"omitempty,notblank,max=256"`
}

// GenerationContextOverride replaces one saved step in automatic project context.
type GenerationContextOverride struct {
	EngineVersionID uuid.UUID       `json:"engineVersionID" validate:"required"`
	StepKey         string          `json:"stepKey"         validate:"required,notblank,max=256"`
	Value           json.RawMessage `json:"value"           validate:"required"`
}

// Terminal reports whether no further generation state can follow.
func (status GenerationStatus) Terminal() bool {
	_, terminal := generationTerminalStatuses[status]

	return terminal
}

// Generation is the owner-scoped lifecycle and volatile schema-validated JSON proposal.
type Generation struct {
	ID          uuid.UUID         `json:"id"`
	Status      GenerationStatus  `json:"status"`
	Target      *GenerationTarget `json:"target,omitempty"`
	Attempt     int32             `json:"attempt"`
	MaxAttempts int32             `json:"maxAttempts"`
	Proposal    json.RawMessage   `json:"proposal"`
	Failure     string            `json:"failure"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	SettledAt   *time.Time        `json:"settledAt"`
	ExpiresAt   *time.Time        `json:"expiresAt"`
}

// GenerationSubmitResult reports whether submission created work or replayed it.
type GenerationSubmitResult struct {
	Generation *Generation
	Created    bool
}
