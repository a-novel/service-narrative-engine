package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

// generationServiceResponse gives cross-field validation a local view of the service-genai envelope.
type generationServiceResponse struct {
	ID              string `validate:"required,uuid"`
	ExpectedID      string `validate:"omitempty,uuid,eqfield=ID"`
	OwnerID         string `validate:"required,uuid"`
	ExpectedOwnerID string `validate:"required,uuid,eqfield=OwnerID"`
	Purpose         string `validate:"required,eq=studio.generation"`
	CreatedAt       string `validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
	UpdatedAt       string `validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
	SettledAt       string `validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	ExpiresAt       string `validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}

// generationStatuses keeps the protobuf enum behind Narrative Engine's string vocabulary.
var generationStatuses = map[servicegenai.GenerationStatus]GenerationStatus{
	servicegenai.GenerationStatusPending:   GenerationStatusPending,
	servicegenai.GenerationStatusRunning:   GenerationStatusRunning,
	servicegenai.GenerationStatusSucceeded: GenerationStatusSucceeded,
	servicegenai.GenerationStatusFailed:    GenerationStatusFailed,
	servicegenai.GenerationStatusAbandoned: GenerationStatusAbandoned,
	servicegenai.GenerationStatusCancelled: GenerationStatusCancelled,
}

var errGenerationProviderFailure = errors.New("generation provider failure")

// mapGeneration validates service-owned metadata and exposes only opaque output and failures.
func mapGeneration(
	ctx context.Context,
	source *servicegenai.Generation,
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

	expectedIDValue := ""
	if expectedID != nil {
		expectedIDValue = expectedID.String()
	}

	response := generationServiceResponse{
		ID:              source.GetId(),
		ExpectedID:      expectedIDValue,
		OwnerID:         source.GetOwnerId(),
		ExpectedOwnerID: expectedOwner.String(),
		Purpose:         source.GetPurpose(),
		CreatedAt:       source.GetCreatedAt(),
		UpdatedAt:       source.GetUpdatedAt(),
		SettledAt:       source.GetSettledAt(),
		ExpiresAt:       source.GetExpiresAt(),
	}

	err := validate.Struct(response)
	if err != nil {
		span.RecordError(err)

		return nil, otel.ReportError(span, ErrGenerationResponseInvalid)
	}

	id, err := uuid.Parse(response.ID)
	if err != nil {
		span.RecordError(err)

		return nil, otel.ReportError(span, ErrGenerationResponseInvalid)
	}

	status, err := mapGenerationStatus(source.GetStatus())
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	createdAt, err := lib.ParseRequiredRFC3339("created_at", response.CreatedAt)
	if err != nil {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: %w", ErrGenerationResponseInvalid, err),
		)
	}

	updatedAt, err := lib.ParseRequiredRFC3339("updated_at", response.UpdatedAt)
	if err != nil {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: %w", ErrGenerationResponseInvalid, err),
		)
	}

	settledAt, err := lib.ParseOptionalRFC3339("settled_at", response.SettledAt)
	if err != nil {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: %w", ErrGenerationResponseInvalid, err),
		)
	}

	expiresAt, err := lib.ParseOptionalRFC3339("expires_at", response.ExpiresAt)
	if err != nil {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: %w", ErrGenerationResponseInvalid, err),
		)
	}

	generation := &Generation{
		ID:          id,
		Status:      status,
		Attempt:     source.GetAttempt(),
		MaxAttempts: source.GetMaxAttempts(),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		SettledAt:   settledAt,
		ExpiresAt:   expiresAt,
	}

	if source.GetError() != "" {
		// service-genai owns secure logging of the original provider failure.
		// Repeating its wire value here could copy private provider details
		// into another service's spans.
		span.RecordError(errGenerationProviderFailure)

		generation.Failure = "generation failed"
	}

	if status == GenerationStatusSucceeded {
		generation.Proposal, err = resolveGenerationProposal(ctx, source.GetOutput())
		if err != nil {
			return nil, otel.ReportError(span, err)
		}
	}

	return otel.ReportSuccess(span, generation), nil
}

// mapGenerationStatus keeps service-genai values behind Narrative Engine's stable vocabulary.
func mapGenerationStatus(status servicegenai.GenerationStatus) (GenerationStatus, error) {
	mapped, known := generationStatuses[status]
	if !known {
		return "", fmt.Errorf("%w: %d", ErrGenerationStatusUnknown, status)
	}

	return mapped, nil
}

// resolveGenerationProposal extracts one bounded JSON value without applying a domain schema.
func resolveGenerationProposal(
	ctx context.Context,
	output []byte,
) ([]byte, error) {
	_, span := otel.Tracer().Start(ctx, "core.resolveGenerationProposal")
	defer span.End()

	text, err := lib.ExtractResponsesOutputText(output)
	if err != nil {
		span.RecordError(err)

		if errors.Is(err, lib.ErrResponsesRefused) {
			return nil, otel.ReportError(span, ErrGenerationRefused)
		}

		return nil, otel.ReportError(span, ErrGenerationOutputInvalid)
	}

	err = lib.ValidateJSON([]byte(text), generationJSONComponentMaxBytes)
	if err != nil {
		span.RecordError(err)

		return nil, otel.ReportError(span, ErrGenerationOutputInvalid)
	}

	return otel.ReportSuccess(span, bytes.Clone([]byte(text))), nil
}
