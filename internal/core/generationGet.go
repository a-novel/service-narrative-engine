package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

// GenerationGetGateway is the domain operation used to retrieve generation state.
type GenerationGetGateway interface {
	// Exec retrieves one owner-scoped generation.
	Exec(ctx context.Context, request *lib.GenerationGetGatewayRequest) (*lib.Generation, error)
}

// GenerationGetRequest identifies one Project-owned generation.
type GenerationGetRequest struct {
	// Actor identifies the authenticated Project owner.
	Actor Actor `validate:"actor"`
	// ProjectID identifies the Project used for authorization.
	ProjectID uuid.UUID `validate:"required"`
	// ID identifies the generation in service-genai.
	ID uuid.UUID `validate:"required"`
}

// GenerationGet reads current state through the generation gateway after Project authorization.
type GenerationGet struct {
	projectAccess ProjectAccessService
	gateway       GenerationGetGateway
}

// NewGenerationGet creates the current-state generation service.
func NewGenerationGet(
	projectAccess ProjectAccessService,
	gateway GenerationGetGateway,
) *GenerationGet {
	return &GenerationGet{projectAccess: projectAccess, gateway: gateway}
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

	response, err := service.gateway.Exec(ctx, &lib.GenerationGetGatewayRequest{
		ID:      request.ID,
		OwnerID: request.Actor.UserID,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	generation, err := mapGeneration(
		ctx,
		response,
		&request.ID,
		request.Actor.UserID,
	)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, generation), nil
}
