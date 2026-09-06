package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

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

	generation := &Generation{
		ID:          source.ID,
		Status:      source.Status,
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

	if source.Status == GenerationStatusSucceeded {
		proposal, err := resolveGenerationProposal(ctx, source.Output)
		if err != nil {
			return nil, otel.ReportError(span, err)
		}

		generation.Proposal = proposal
	}

	return otel.ReportSuccess(span, generation), nil
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

	return otel.ReportSuccess(span, []byte(output)), nil
}
