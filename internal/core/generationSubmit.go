package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

// GenerationSubmitRequest contains the complete client-composed generation.
type GenerationSubmitRequest struct {
	// Actor identifies the authenticated Project owner.
	Actor Actor `validate:"actor"`
	// ProjectID scopes authorization and idempotency to one Project.
	ProjectID uuid.UUID `validate:"required"`
	// IdempotencyKey identifies one caller retry sequence within the Project.
	IdempotencyKey string `validate:"required,notblank,max=256"`
	// Instructions contains the caller-controlled generation guidance.
	Instructions string `validate:"required,notblank,max=32768"`
	// Input is the caller-controlled partial value to complete.
	Input json.RawMessage `validate:"required"`
	// Context is the caller-controlled source material for the generation.
	Context json.RawMessage `validate:"required"`
	// OutputSchema is the caller-controlled provider output contract.
	OutputSchema json.RawMessage `validate:"required"`
}

// GenerationSubmit authorizes and forwards one client-composed generation.
type GenerationSubmit struct {
	projectAccess ProjectAccessService
	genai         servicegenai.Client
}

// NewGenerationSubmit creates the thin Narrative generation adapter.
func NewGenerationSubmit(
	projectAccess ProjectAccessService,
	genai servicegenai.Client,
) *GenerationSubmit {
	return &GenerationSubmit{projectAccess: projectAccess, genai: genai}
}

// Exec submits one priced generation or attaches to the caller's existing work.
func (service *GenerationSubmit) Exec(
	ctx context.Context,
	request *GenerationSubmitRequest,
) (*GenerationSubmitResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationSubmit")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.Actor.UserID.String()),
	)

	_, err = service.projectAccess.Exec(ctx, &ProjectAccessRequest{
		Actor:     request.Actor,
		ProjectID: request.ProjectID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("access Project: %w", err))
	}

	err = validateGenerationJSON("input", request.Input)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	err = validateGenerationJSON("context", request.Context)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	err = validateGenerationJSON("output schema", request.OutputSchema)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	payload, err := buildGenerationPayload(
		request.Actor.UserID,
		request.Instructions,
		request.Input,
		request.Context,
		request.OutputSchema,
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("build generation payload: %w", err))
	}

	if len(payload) > generationProviderRequestMaxBytes {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: provider request exceeds the size limit", ErrInvalidRequest),
		)
	}

	idempotencyKey, err := deriveGenerationIdempotencyKey(
		request.IdempotencyKey,
		request.ProjectID,
	)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	response, err := service.genai.GenerationSubmit(ctx, &servicegenai.GenerationSubmitRequest{
		OwnerId:        request.Actor.UserID.String(),
		Purpose:        GenerationPurposeStudio,
		IdempotencyKey: idempotencyKey,
		Request:        payload,
		MaxAttempts:    generationMaxAttempts,
	})
	if status.Code(err) == codes.AlreadyExists {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %w", ErrGenerationConflict, err))
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("submit generation: %w", err))
	}

	if response == nil {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: missing submit response", ErrGenerationResponseInvalid),
		)
	}

	generation, err := mapGeneration(
		ctx,
		response.GetGeneration(),
		nil,
		request.Actor.UserID,
	)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(
		attribute.String("generation.id", generation.ID.String()),
		attribute.Bool("generation.created", response.GetCreated()),
	)

	return otel.ReportSuccess(span, &GenerationSubmitResult{
		Generation: generation,
		Created:    response.GetCreated(),
	}), nil
}

// validateGenerationJSON applies the transport ceiling without interpreting caller JSON.
func validateGenerationJSON(name string, value json.RawMessage) error {
	err := lib.ValidateJSON(value, generationJSONComponentMaxBytes)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrInvalidRequest, name, err)
	}

	return nil
}
