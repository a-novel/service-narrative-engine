package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

// GenerationGetRequest identifies one Project-owned generation.
type GenerationGetRequest struct {
	// Actor identifies the authenticated Project owner.
	Actor Actor `validate:"actor"`
	// ProjectID identifies the Project used for authorization.
	ProjectID uuid.UUID `validate:"required"`
	// ID identifies the generation in service-genai.
	ID uuid.UUID `validate:"required"`
}

// GenerationGet reads current state directly from service-genai after Project authorization.
type GenerationGet struct {
	projectAccess ProjectAccessService
	genai         servicegenai.Client
}

// NewGenerationGet creates the current-state generation service.
func NewGenerationGet(
	projectAccess ProjectAccessService,
	genai servicegenai.Client,
) *GenerationGet {
	return &GenerationGet{projectAccess: projectAccess, genai: genai}
}

// Exec returns current lifecycle state and an opaque JSON proposal on success.
func (service *GenerationGet) Exec(
	ctx context.Context,
	request *GenerationGetRequest,
) (*Generation, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationGet")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("generation.id", request.ID.String()),
		attribute.String("project.owner_id", request.Actor.UserID.String()),
	)

	_, err = service.projectAccess.Exec(ctx, &ProjectAccessRequest{
		Actor:     request.Actor,
		ProjectID: request.ProjectID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("access Project: %w", err))
	}

	response, err := service.genai.GenerationGet(ctx, &servicegenai.GenerationGetRequest{
		Id:      request.ID.String(),
		OwnerId: request.Actor.UserID.String(),
	})
	if status.Code(err) == codes.NotFound {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %w", ErrGenerationNotFound, err))
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get generation: %w", err))
	}

	if response == nil {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: missing get response", ErrGenerationResponseInvalid),
		)
	}

	generation, err := mapGeneration(
		ctx,
		response.GetGeneration(),
		&request.ID,
		request.Actor.UserID,
	)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, generation), nil
}
