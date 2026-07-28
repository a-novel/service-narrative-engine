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

// EngineVersionSelectDao retrieves immutable definitions used to validate
// retained generation output.
type EngineVersionSelectDao interface {
	Exec(ctx context.Context, request *dao.EngineVersionSelectRequest) (*dao.EngineVersion, error)
}

type generationOutputContext struct {
	engineVersionID uuid.UUID
	step            *engineStepDefinition
}

type generationOutputEnvelope struct {
	EngineVersionID string          `json:"engineVersionID"`
	StepKey         string          `json:"stepKey"`
	Value           json.RawMessage `json:"value"`
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
		return nil, fmt.Errorf("%w: missing generation", ErrGenerationResponseInvalid)
	}

	id, err := uuid.Parse(source.GetId())
	if err != nil {
		return nil, fmt.Errorf("%w: parse id: %w", ErrGenerationResponseInvalid, err)
	}

	if expectedID != nil && id != *expectedID {
		return nil, fmt.Errorf("%w: expected id %s, got %s", ErrGenerationResponseInvalid, expectedID, id)
	}

	ownerID, err := uuid.Parse(source.GetOwnerId())
	if err != nil {
		return nil, fmt.Errorf("%w: parse owner id: %w", ErrGenerationResponseInvalid, err)
	}

	if ownerID != expectedOwner {
		return nil, fmt.Errorf("%w: owner mismatch", ErrGenerationResponseInvalid)
	}

	if source.GetPurpose() != GenerationPurposeStudio {
		return nil, fmt.Errorf("%w: purpose mismatch", ErrGenerationResponseInvalid)
	}

	status, err := mapGenerationStatus(source.GetStatus())
	if err != nil {
		return nil, err
	}

	createdAt, err := parseRequiredGenerationTime("created_at", source.GetCreatedAt())
	if err != nil {
		return nil, err
	}

	updatedAt, err := parseRequiredGenerationTime("updated_at", source.GetUpdatedAt())
	if err != nil {
		return nil, err
	}

	settledAt, err := parseOptionalGenerationTime("settled_at", source.GetSettledAt())
	if err != nil {
		return nil, err
	}

	expiresAt, err := parseOptionalGenerationTime("expires_at", source.GetExpiresAt())
	if err != nil {
		return nil, err
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

	if status == GenerationStatusSucceeded {
		generation.Proposal, err = resolveGenerationProposal(
			ctx,
			engineVersionDao,
			source.GetOutput(),
			expectedContext,
		)
		if err != nil {
			return nil, err
		}
	}

	return generation, nil
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
) (*ManuscriptValue, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.resolveGenerationProposal")
	defer span.End()

	text, err := extractResponsesOutputText(output)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGenerationOutputInvalid, err)
	}

	var envelope generationOutputEnvelope

	decoder := json.NewDecoder(bytes.NewReader([]byte(text)))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&envelope)
	if err != nil {
		return nil, fmt.Errorf("%w: decode envelope: %w", ErrGenerationOutputInvalid, err)
	}

	err = ensureJSONEOF(decoder)
	if err != nil {
		return nil, fmt.Errorf("%w: decode envelope: %w", ErrGenerationOutputInvalid, err)
	}

	engineVersionID, err := uuid.Parse(envelope.EngineVersionID)
	if err != nil {
		return nil, fmt.Errorf("%w: parse engine version id: %w", ErrGenerationOutputInvalid, err)
	}

	if strings.TrimSpace(envelope.StepKey) == "" || len(envelope.Value) == 0 {
		return nil, fmt.Errorf("%w: incomplete envelope", ErrGenerationOutputInvalid)
	}

	var step *engineStepDefinition

	if expectedContext != nil {
		if engineVersionID != expectedContext.engineVersionID || envelope.StepKey != expectedContext.step.Key {
			return nil, fmt.Errorf("%w: envelope context mismatch", ErrGenerationOutputInvalid)
		}

		step = expectedContext.step
	} else {
		engineVersion, selectErr := engineVersionDao.Exec(ctx, &dao.EngineVersionSelectRequest{
			ID: engineVersionID,
		})
		if selectErr != nil {
			if errors.Is(selectErr, dao.ErrEngineVersionSelectNotFound) {
				selectErr = errors.Join(selectErr, ErrEngineVersionNotFound)
			}

			return nil, fmt.Errorf("%w: select engine version: %w", ErrGenerationOutputInvalid, selectErr)
		}

		step, err = selectEngineStep(engineVersion.Definition, envelope.StepKey)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrGenerationOutputInvalid, err)
		}
	}

	err = step.validateValue(envelope.Value)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGenerationOutputInvalid, err)
	}

	var proposal ManuscriptValue

	err = json.Unmarshal(envelope.Value, &proposal)
	if err != nil {
		return nil, fmt.Errorf("%w: decode Manuscript: %w", ErrGenerationOutputInvalid, err)
	}

	return &proposal, nil
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
		return "", fmt.Errorf("decode Responses output: %w", err)
	}

	if response.OutputText != "" {
		return response.OutputText, nil
	}

	var text strings.Builder

	for _, item := range response.Output {
		for _, content := range item.Content {
			switch content.Type {
			case "output_text":
				text.WriteString(content.Text)
			case "refusal":
				return "", fmt.Errorf("%w: %s", ErrGenerationRefused, content.Refusal)
			}
		}
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
