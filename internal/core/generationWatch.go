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

// GenerationWatchRequest identifies one owner-scoped generation to await.
type GenerationWatchRequest struct {
	Actor Actor     `validate:"actor"`
	ID    uuid.UUID `validate:"required"`
}

// GenerationWatch waits on service-genai's resumable state stream.
type GenerationWatch struct {
	engineVersionDao EngineVersionSelectDao
	genai            servicegenai.Client
}

// NewGenerationWatch creates the low-latency generation wait service.
func NewGenerationWatch(
	engineVersionDao EngineVersionSelectDao,
	genai servicegenai.Client,
) *GenerationWatch {
	return &GenerationWatch{engineVersionDao: engineVersionDao, genai: genai}
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
		attribute.String("generation.id", request.ID.String()),
		attribute.String("generation.owner_id", request.Actor.UserID.String()),
	)

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
			return nil, otel.ReportError(span, fmt.Errorf("%w: %w", ErrGenerationNotFound, receiveErr))
		}

		if receiveErr != nil {
			return nil, otel.ReportError(span, fmt.Errorf("receive generation: %w", receiveErr))
		}

		if response == nil {
			return nil, otel.ReportError(span, fmt.Errorf("%w: missing watch response", ErrGenerationResponseInvalid))
		}

		generation, mapErr := mapGeneration(
			ctx,
			service.engineVersionDao,
			response.GetGeneration(),
			&request.ID,
			request.Actor.UserID,
			nil,
		)
		if mapErr != nil {
			return nil, otel.ReportError(span, mapErr)
		}

		if generation.Status.Terminal() {
			return otel.ReportSuccess(span, generation), nil
		}
	}
}
