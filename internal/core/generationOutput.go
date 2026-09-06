package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

var generationStatuses = map[lib.GenerationStatus]GenerationStatus{
	lib.GenerationStatusPending:   GenerationStatusPending,
	lib.GenerationStatusRunning:   GenerationStatusRunning,
	lib.GenerationStatusSucceeded: GenerationStatusSucceeded,
	lib.GenerationStatusFailed:    GenerationStatusFailed,
	lib.GenerationStatusAbandoned: GenerationStatusAbandoned,
	lib.GenerationStatusCancelled: GenerationStatusCancelled,
}

var errGenerationProviderFailure = errors.New("generation provider failure")

// mapGeneration validates service-owned metadata and exposes only opaque output and failures.
func mapGeneration(
	ctx context.Context,
	source *lib.Generation,
	expectedID *uuid.UUID,
	expectedOwner uuid.UUID,
) (*Generation, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.mapGeneration")
	defer span.End()

	if source == nil {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: missing generation", ErrGenerationResponseInvalid),
		)
	}

	if source.ID == uuid.Nil ||
		source.OwnerID == uuid.Nil ||
		source.OwnerID != expectedOwner ||
		source.Purpose != GenerationPurposeStudio ||
		source.CreatedAt.IsZero() ||
		source.UpdatedAt.IsZero() ||
		(expectedID != nil && source.ID != *expectedID) {
		return nil, otel.ReportError(span, ErrGenerationResponseInvalid)
	}

	status, err := mapGenerationStatus(source.Status)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	generation := &Generation{
		ID:          source.ID,
		Status:      status,
		Attempt:     source.Attempt,
		MaxAttempts: source.MaxAttempts,
		CreatedAt:   source.CreatedAt,
		UpdatedAt:   source.UpdatedAt,
		SettledAt:   source.SettledAt,
		ExpiresAt:   source.ExpiresAt,
	}

	if source.Failed {
		// The gateway withholds provider details and exposes only their presence.
		span.RecordError(errGenerationProviderFailure)

		generation.Failure = "generation failed"
	}

	if status == GenerationStatusSucceeded {
		generation.Proposal, err = resolveGenerationProposal(ctx, source.Output)
		if err != nil {
			return nil, otel.ReportError(span, err)
		}
	}

	return otel.ReportSuccess(span, generation), nil
}

// mapGenerationStatus keeps gateway values behind Narrative Engine's stable vocabulary.
func mapGenerationStatus(status lib.GenerationStatus) (GenerationStatus, error) {
	mapped, known := generationStatuses[status]
	if !known {
		return "", fmt.Errorf("%w: %s", ErrGenerationStatusUnknown, status)
	}

	return mapped, nil
}

// resolveGenerationProposal extracts one bounded JSON value without applying a domain schema.
func resolveGenerationProposal(
	ctx context.Context,
	output string,
) ([]byte, error) {
	_, span := otel.Tracer().Start(ctx, "core.resolveGenerationProposal")
	defer span.End()

	err := lib.ValidateJSON([]byte(output), generationJSONComponentMaxBytes)
	if err != nil {
		span.RecordError(err)

		return nil, otel.ReportError(span, ErrGenerationOutputInvalid)
	}

	return otel.ReportSuccess(span, bytes.Clone([]byte(output))), nil
}
