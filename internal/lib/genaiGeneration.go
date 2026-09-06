package lib

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	servicegenai "github.com/a-novel/service-genai/pkg/go"
)

var (
	// ErrGenerationConflict is returned when an idempotency key identifies different work.
	ErrGenerationConflict = errors.New("generation idempotency conflict")
	// ErrGenerationNotFound is returned when service-genai has no matching owner-scoped generation.
	ErrGenerationNotFound = errors.New("generation not found")
	// ErrGenerationOutputInvalid is returned when provider output cannot be exposed as generated text.
	ErrGenerationOutputInvalid = errors.New("invalid generation output")
	// ErrGenerationRefused is returned when the provider refuses a generation.
	ErrGenerationRefused = errors.New("generation refused")
	// ErrGenerationResponseInvalid is returned when service-genai breaks its response contract.
	ErrGenerationResponseInvalid = errors.New("invalid generation response")
	// ErrGenerationStatusUnknown is returned for an unsupported service-genai lifecycle status.
	ErrGenerationStatusUnknown = errors.New("unknown generation status")
)

var generationStatuses = map[servicegenai.GenerationStatus]string{
	servicegenai.GenerationStatusPending:   "pending",
	servicegenai.GenerationStatusRunning:   "running",
	servicegenai.GenerationStatusSucceeded: "succeeded",
	servicegenai.GenerationStatusFailed:    "failed",
	servicegenai.GenerationStatusAbandoned: "abandoned",
	servicegenai.GenerationStatusCancelled: "cancelled",
}

// Generation is a decoded service-genai snapshot awaiting workflow validation.
type Generation struct {
	// ID identifies the generation in service-genai.
	ID uuid.UUID
	// OwnerID identifies the owner used to scope the generation.
	OwnerID uuid.UUID
	// Purpose identifies the workflow that submitted the generation.
	Purpose string
	// Status is service-genai's lifecycle state normalized to a lowercase name.
	Status string
	// Attempt is the number of provider runs already started.
	Attempt int32
	// MaxAttempts is the maximum number of provider runs allowed.
	MaxAttempts int32
	// Output contains generated text after provider-envelope extraction.
	Output string
	// Failed reports whether service-genai recorded a private provider failure.
	Failed bool
	// CreatedAt records when service-genai accepted the generation.
	CreatedAt time.Time
	// UpdatedAt records the latest lifecycle change.
	UpdatedAt time.Time
	// SettledAt records when the generation reached a terminal state.
	SettledAt *time.Time
	// ExpiresAt records when service-genai may purge the generation.
	ExpiresAt *time.Time
}

func mapGenAIGeneration(source *servicegenai.Generation) (*Generation, error) {
	if source == nil {
		return nil, fmt.Errorf("%w: missing generation", ErrGenerationResponseInvalid)
	}

	id, err := uuid.Parse(source.GetId())
	if err != nil {
		return nil, fmt.Errorf("%w: parse generation id: %w", ErrGenerationResponseInvalid, err)
	}

	ownerID, err := uuid.Parse(source.GetOwnerId())
	if err != nil {
		return nil, fmt.Errorf("%w: parse generation owner id: %w", ErrGenerationResponseInvalid, err)
	}

	status, known := generationStatuses[source.GetStatus()]
	if !known {
		return nil, fmt.Errorf("%w: %d", ErrGenerationStatusUnknown, source.GetStatus())
	}

	createdAt, err := ParseRequiredRFC3339("created_at", source.GetCreatedAt())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGenerationResponseInvalid, err)
	}

	updatedAt, err := ParseRequiredRFC3339("updated_at", source.GetUpdatedAt())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGenerationResponseInvalid, err)
	}

	settledAt, err := ParseOptionalRFC3339("settled_at", source.GetSettledAt())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGenerationResponseInvalid, err)
	}

	expiresAt, err := ParseOptionalRFC3339("expires_at", source.GetExpiresAt())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGenerationResponseInvalid, err)
	}

	generation := &Generation{
		ID:          id,
		OwnerID:     ownerID,
		Purpose:     source.GetPurpose(),
		Status:      status,
		Attempt:     source.GetAttempt(),
		MaxAttempts: source.GetMaxAttempts(),
		Failed:      source.GetError() != "",
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		SettledAt:   settledAt,
		ExpiresAt:   expiresAt,
	}

	if source.GetStatus() == servicegenai.GenerationStatusSucceeded {
		generation.Output, err = servicegenai.ExtractResponsesOutputText(source.GetOutput())
		if errors.Is(err, servicegenai.ErrResponsesRefused) {
			return nil, fmt.Errorf("%w: %w", ErrGenerationRefused, err)
		}

		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrGenerationOutputInvalid, err)
		}
	}

	return generation, nil
}
