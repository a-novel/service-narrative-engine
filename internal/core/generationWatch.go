package core

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

// GenerationWatchGateway is the domain operation used to stream generation state.
type GenerationWatchGateway interface {
	// Exec opens one owner-scoped generation stream.
	Exec(
		ctx context.Context,
		request *lib.GenerationWatchGatewayRequest,
	) (lib.GenerationWatchGatewayStream, error)
}

// GenerationWatchRequest identifies one Project-owned generation to await.
type GenerationWatchRequest struct {
	// Actor identifies the authenticated Project owner.
	Actor Actor `validate:"actor"`
	// ProjectID identifies the Project used for authorization.
	ProjectID uuid.UUID `validate:"required"`
	// ID identifies the generation in service-genai.
	ID uuid.UUID `validate:"required"`
}

// GenerationWatch waits on the generation gateway after Project authorization.
type GenerationWatch struct {
	projectAccess ProjectAccessService
	gateway       GenerationWatchGateway
}

// NewGenerationWatch creates the low-latency generation wait service.
func NewGenerationWatch(
	projectAccess ProjectAccessService,
	gateway GenerationWatchGateway,
) *GenerationWatch {
	return &GenerationWatch{projectAccess: projectAccess, gateway: gateway}
}

// Exec consumes snapshots until service-genai reports a terminal state.
func (service *GenerationWatch) Exec(
	ctx context.Context,
	request *GenerationWatchRequest,
) (*Generation, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationWatch")
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

	stream, err := service.gateway.Exec(ctx, &lib.GenerationWatchGatewayRequest{
		ID:      request.ID,
		OwnerID: request.Actor.UserID,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	if stream == nil {
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: missing watch stream", ErrGenerationResponseInvalid),
		)
	}
	defer stream.Close()

	for {
		response, receiveErr := stream.Recv()
		if errors.Is(receiveErr, io.EOF) {
			return nil, otel.ReportError(span, ErrGenerationWatchClosed)
		}

		if receiveErr != nil {
			return nil, otel.ReportError(span, receiveErr)
		}

		generation, mapErr := mapGeneration(
			ctx,
			response,
			&request.ID,
			request.Actor.UserID,
		)
		if mapErr != nil {
			return nil, otel.ReportError(span, mapErr)
		}

		if generation.Status.Terminal() {
			return otel.ReportSuccess(span, generation), nil
		}
	}
}
