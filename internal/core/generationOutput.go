package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/lib"
)

// EngineVersionSelectDao retrieves immutable definitions used to validate
// retained generation output.
type EngineVersionSelectDao interface {
	Exec(ctx context.Context, request *dao.EngineVersionSelectRequest) (*dao.EngineVersion, error)
}

// generationOutputContext reuses the submit-time contract when a submission
// response is mapped before its target can be read from retained output.
type generationOutputContext struct {
	definition *generationTargetDefinition
}

// generationOutputEnvelope pins generated content to the target identity that
// selects its complete local validator.
type generationOutputEnvelope struct {
	TargetKind      GenerationTargetKind `json:"targetKind"      validate:"required,oneof=idea step manuscript"`
	EngineVersionID string               `json:"engineVersionID" validate:"omitempty,uuid"`
	StepKey         string               `json:"stepKey"         validate:"omitempty,notblank,max=256"`
	Value           json.RawMessage      `json:"value"           validate:"required"`
}

// generationServiceResponse gives validator cross-field checks a local view of
// the owner-scoped service-genai envelope.
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

var generationStatuses = map[servicegenai.GenerationStatus]GenerationStatus{
	servicegenai.GenerationStatusPending:   GenerationStatusPending,
	servicegenai.GenerationStatusRunning:   GenerationStatusRunning,
	servicegenai.GenerationStatusSucceeded: GenerationStatusSucceeded,
	servicegenai.GenerationStatusFailed:    GenerationStatusFailed,
	servicegenai.GenerationStatusAbandoned: GenerationStatusAbandoned,
	servicegenai.GenerationStatusCancelled: GenerationStatusCancelled,
}

// mapGeneration validates the service-genai envelope before exposing stable
// Narrative lifecycle state or schema-validated output.
func mapGeneration(
	ctx context.Context,
	engineVersionDao EngineVersionSelectDao,
	source *servicegenai.Generation,
	expectedID *uuid.UUID,
	expectedOwner uuid.UUID,
	expectedContext *generationOutputContext,
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
		Failure:     source.GetError(),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		SettledAt:   settledAt,
		ExpiresAt:   expiresAt,
	}

	if expectedContext != nil {
		if expectedContext.definition == nil {
			return nil, otel.ReportError(
				span,
				fmt.Errorf("%w: missing expected target", ErrGenerationResponseInvalid),
			)
		}

		target := expectedContext.definition.Target
		generation.Target = &target
	}

	if status == GenerationStatusSucceeded {
		var target GenerationTarget

		generation.Proposal, target, err = resolveGenerationProposal(
			ctx,
			engineVersionDao,
			source.GetOutput(),
			expectedContext,
		)
		if err != nil {
			return nil, otel.ReportError(span, err)
		}

		generation.Target = &target
	}

	return otel.ReportSuccess(span, generation), nil
}

// mapGenerationStatus keeps service-genai values behind Narrative Engine's
// stable lifecycle vocabulary.
func mapGenerationStatus(status servicegenai.GenerationStatus) (GenerationStatus, error) {
	mapped, known := generationStatuses[status]
	if !known {
		return "", fmt.Errorf("%w: %d", ErrGenerationStatusUnknown, status)
	}

	return mapped, nil
}

// resolveGenerationProposal extracts the provider envelope, resolves its
// target contract, and validates the complete proposal before returning it.
func resolveGenerationProposal(
	ctx context.Context,
	engineVersionDao EngineVersionSelectDao,
	output json.RawMessage,
	expectedContext *generationOutputContext,
) (json.RawMessage, GenerationTarget, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.resolveGenerationProposal")
	defer span.End()

	var noTarget GenerationTarget

	text, err := lib.ExtractResponsesOutputText(output)
	if err != nil {
		if errors.Is(err, lib.ErrResponsesRefused) {
			err = ErrGenerationRefused
		}

		return nil, noTarget, otel.ReportError(
			span,
			fmt.Errorf("%w: %w", ErrGenerationOutputInvalid, err),
		)
	}

	var envelope generationOutputEnvelope

	err = lib.DecodeJSONStrict([]byte(text), &envelope)
	if err != nil {
		span.RecordError(err)

		return nil, noTarget, otel.ReportError(span, ErrGenerationOutputInvalid)
	}

	err = validate.Struct(envelope)
	if err != nil {
		span.RecordError(err)

		return nil, noTarget, otel.ReportError(span, ErrGenerationOutputInvalid)
	}

	target, err := generationTargetFromEnvelope(&envelope)
	if err != nil {
		return nil, noTarget, otel.ReportError(span, ErrGenerationOutputInvalid)
	}

	var definition *generationTargetDefinition

	if expectedContext != nil {
		if expectedContext.definition == nil || target != expectedContext.definition.Target {
			return nil, noTarget, otel.ReportError(
				span,
				fmt.Errorf("%w: envelope context mismatch", ErrGenerationOutputInvalid),
			)
		}

		definition = expectedContext.definition
	} else {
		definition, err = loadGenerationTarget(ctx, engineVersionDao, target)
		if err != nil {
			switch {
			case errors.Is(err, ErrEngineVersionNotFound):
				err = errors.Join(ErrGenerationOutputInvalid, ErrEngineVersionNotFound)
			case errors.Is(err, ErrEngineStepNotFound), errors.Is(err, ErrInvalidRequest):
				err = ErrGenerationOutputInvalid
			default:
				err = fmt.Errorf("%w: load target: %w", ErrGenerationOutputInvalid, err)
			}

			return nil, noTarget, otel.ReportError(span, err)
		}
	}

	err = definition.validateComplete(envelope.Value)
	if err != nil {
		return nil, noTarget, otel.ReportError(
			span,
			fmt.Errorf("%w: %w", ErrGenerationOutputInvalid, err),
		)
	}

	otel.ReportSuccessNoContent(span)

	return envelope.Value, target, nil
}

// generationTargetFromEnvelope converts validated wire strings into the
// target identity used by local schema selection.
func generationTargetFromEnvelope(envelope *generationOutputEnvelope) (GenerationTarget, error) {
	target := GenerationTarget{
		Kind:    envelope.TargetKind,
		StepKey: envelope.StepKey,
	}

	if target.Kind == GenerationTargetKindStep {
		engineVersionID, err := uuid.Parse(envelope.EngineVersionID)
		if err != nil {
			return GenerationTarget{}, fmt.Errorf("parse engine version id: %w", err)
		}

		target.EngineVersionID = engineVersionID
	}

	err := validate.Struct(target)
	if err != nil {
		return GenerationTarget{}, err
	}

	return target, nil
}
