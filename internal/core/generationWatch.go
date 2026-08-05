package core

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

// GenerationWatchRequest identifies one Project-owned generation to await.
type GenerationWatchRequest struct {
	Actor     Actor     `validate:"actor"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`
}

// GenerationWatch waits on service-genai after Project authorization.
type GenerationWatch struct {
	projectAccess ProjectAccessService
	genai         servicegenai.Client
}

// NewGenerationWatch creates the low-latency generation wait service.
func NewGenerationWatch(
	projectAccess ProjectAccessService,
	genai servicegenai.Client,
) *GenerationWatch {
	return &GenerationWatch{projectAccess: projectAccess, genai: genai}
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

	stream, err := service.genai.GenerationWatch(ctx, &servicegenai.GenerationWatchRequest{
		Id:      request.ID.String(),
		OwnerId: request.Actor.UserID.String(),
	})
	if status.Code(err) == codes.NotFound {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %w", ErrGenerationNotFound, err))
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("watch generation: %w", err))
	}

	for {
		response, receiveErr := stream.Recv()
		if errors.Is(receiveErr, io.EOF) {
			return nil, otel.ReportError(span, ErrGenerationWatchClosed)
		}

		if status.Code(receiveErr) == codes.NotFound {
			return nil, otel.ReportError(
				span,
				fmt.Errorf("%w: %w", ErrGenerationNotFound, receiveErr),
			)
		}

		if receiveErr != nil {
			return nil, otel.ReportError(span, fmt.Errorf("receive generation: %w", receiveErr))
		}

		if response == nil {
			return nil, otel.ReportError(
				span,
				fmt.Errorf("%w: missing watch response", ErrGenerationResponseInvalid),
			)
		}

		generation, mapErr := mapGeneration(
			ctx,
			response.GetGeneration(),
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
