package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	errGenerationStaticTargetStep  = errors.New("static target identifies an Engine step")
	errGenerationTargetKindUnknown = errors.New("unknown target kind")
)

// EngineVersionSelectDao retrieves immutable definitions used to validate
// retained generation output.
type EngineVersionSelectDao interface {
	Exec(ctx context.Context, request *dao.EngineVersionSelectRequest) (*dao.EngineVersion, error)
}

type generationOutputContext struct {
	definition *generationTargetDefinition
}

type generationOutputEnvelope struct {
	TargetKind      GenerationTargetKind `json:"targetKind"`
	EngineVersionID string               `json:"engineVersionID"`
	StepKey         string               `json:"stepKey"`
	Value           json.RawMessage      `json:"value"`
}

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

	id, err := uuid.Parse(source.GetId())
	if err != nil {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: parse id: %w", ErrGenerationResponseInvalid, err),
		)
	}

	if expectedID != nil && id != *expectedID {
		return nil, otel.ReportError(span, fmt.Errorf(
			"%w: expected id %s, got %s",
			ErrGenerationResponseInvalid,
			expectedID,
			id,
		))
	}

	ownerID, err := uuid.Parse(source.GetOwnerId())
	if err != nil {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: parse owner id: %w", ErrGenerationResponseInvalid, err),
		)
	}

	if ownerID != expectedOwner {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: owner mismatch", ErrGenerationResponseInvalid),
		)
	}

	if source.GetPurpose() != GenerationPurposeStudio {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: purpose mismatch", ErrGenerationResponseInvalid),
		)
	}

	status, err := mapGenerationStatus(source.GetStatus())
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	createdAt, err := parseRequiredGenerationTime("created_at", source.GetCreatedAt())
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	updatedAt, err := parseRequiredGenerationTime("updated_at", source.GetUpdatedAt())
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	settledAt, err := parseOptionalGenerationTime("settled_at", source.GetSettledAt())
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	expiresAt, err := parseOptionalGenerationTime("expires_at", source.GetExpiresAt())
	if err != nil {
		return nil, otel.ReportError(span, err)
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

func mapGenerationStatus(status servicegenai.GenerationStatus) (GenerationStatus, error) {
	switch status {
	case servicegenai.GenerationStatusPending:
		return GenerationStatusPending, nil
	case servicegenai.GenerationStatusRunning:
		return GenerationStatusRunning, nil
	case servicegenai.GenerationStatusSucceeded:
		return GenerationStatusSucceeded, nil
	case servicegenai.GenerationStatusFailed:
		return GenerationStatusFailed, nil
	case servicegenai.GenerationStatusAbandoned:
		return GenerationStatusAbandoned, nil
	case servicegenai.GenerationStatusCancelled:
		return GenerationStatusCancelled, nil
	default:
		return "", fmt.Errorf("%w: %d", ErrGenerationStatusUnknown, status)
	}
}

func parseRequiredGenerationTime(name string, value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("%w: %s is empty", ErrGenerationResponseInvalid, name)
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: parse %s: %w", ErrGenerationResponseInvalid, name, err)
	}

	return parsed, nil
}

func parseOptionalGenerationTime(name string, value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}

	parsed, err := parseRequiredGenerationTime(name, value)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func resolveGenerationProposal(
	ctx context.Context,
	engineVersionDao EngineVersionSelectDao,
	output json.RawMessage,
	expectedContext *generationOutputContext,
) (json.RawMessage, GenerationTarget, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.resolveGenerationProposal")
	defer span.End()

	var noTarget GenerationTarget

	text, err := extractResponsesOutputText(output)
	if err != nil {
		return nil, noTarget, otel.ReportError(
			span,
			fmt.Errorf("%w: %w", ErrGenerationOutputInvalid, err),
		)
	}

	var envelope generationOutputEnvelope

	decoder := json.NewDecoder(bytes.NewReader([]byte(text)))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&envelope)
	if err != nil {
		return nil, noTarget, otel.ReportError(span, ErrGenerationOutputInvalid)
	}

	err = ensureJSONEOF(decoder)
	if err != nil {
		return nil, noTarget, otel.ReportError(span, ErrGenerationOutputInvalid)
	}

	if len(envelope.Value) == 0 {
		return nil, noTarget, otel.ReportError(
			span,
			fmt.Errorf("%w: incomplete envelope", ErrGenerationOutputInvalid),
		)
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

func generationTargetFromEnvelope(envelope *generationOutputEnvelope) (GenerationTarget, error) {
	target := GenerationTarget{
		Kind:    envelope.TargetKind,
		StepKey: envelope.StepKey,
	}

	switch target.Kind {
	case GenerationTargetKindStep:
		engineVersionID, err := uuid.Parse(envelope.EngineVersionID)
		if err != nil {
			return GenerationTarget{}, fmt.Errorf("parse engine version id: %w", err)
		}

		target.EngineVersionID = engineVersionID
	case GenerationTargetKindIdea, GenerationTargetKindManuscript:
		if envelope.EngineVersionID != "" || envelope.StepKey != "" {
			return GenerationTarget{}, errGenerationStaticTargetStep
		}
	default:
		return GenerationTarget{}, fmt.Errorf("%w: %q", errGenerationTargetKindUnknown, target.Kind)
	}

	err := validateGenerationTarget(target)
	if err != nil {
		return GenerationTarget{}, err
	}

	return target, nil
}

func extractResponsesOutputText(output json.RawMessage) (string, error) {
	if len(output) == 0 {
		return "", errGenerationOutputEmpty
	}

	var response struct {
		// OpenAI owns this snake_case field.
		//nolint:tagliatelle
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				Refusal string `json:"refusal"`
			} `json:"content"`
		} `json:"output"`
	}

	err := json.Unmarshal(output, &response)
	if err != nil {
		return "", errGenerationOutputMalformed
	}

	// Scan before reading the aggregate: a response carrying both text and a
	// refusal is a refusal, and reporting it as malformed output would name the
	// wrong cause.
	var text strings.Builder

	for _, item := range response.Output {
		for _, content := range item.Content {
			switch content.Type {
			case "output_text":
				text.WriteString(content.Text)
			case "refusal":
				return "", ErrGenerationRefused
			}
		}
	}

	if response.OutputText != "" {
		return response.OutputText, nil
	}

	if text.Len() == 0 {
		return "", errGenerationOutputTextMissing
	}

	return text.String(), nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any

	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}

	if err != nil {
		return err
	}

	return errGenerationOutputMultiple
}
