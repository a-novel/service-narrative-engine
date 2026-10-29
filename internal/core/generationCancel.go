package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

// GenerationCancelServiceGenai cancels a genai generation of the caller.
type GenerationCancelServiceGenai interface {
	GenerationCancel(
		ctx context.Context, req *servicegenai.GenerationCancelRequest, opts ...grpc.CallOption,
	) (*servicegenai.GenerationCancelResponse, error)
}

// GenerationCancelRequest names the genai generation to cancel, and its caller.
type GenerationCancelRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`
}

// GenerationCancel stops one call of a generation through genai, as the caller, so it stops costing. A call
// that settled first comes back as it stands, its output included, so a retried cancel succeeds.
type GenerationCancel struct {
	genai GenerationCancelServiceGenai
}

func NewGenerationCancel(genai GenerationCancelServiceGenai) *GenerationCancel {
	return &GenerationCancel{genai: genai}
}

func (service *GenerationCancel) Exec(
	ctx context.Context, request *GenerationCancelRequest,
) (*GenerationResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationCancel")
	defer span.End()

	span.SetAttributes(
		attribute.String("generation.id", request.ID.String()),
		attribute.String("generation.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	response, err := service.genai.GenerationCancel(ctx, &servicegenai.GenerationCancelRequest{
		Id: request.ID.String(), OwnerId: request.OwnerID.String(),
	})
	if status.Code(err) == codes.NotFound {
		err = errors.Join(err, ErrGenerationNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("genai cancel: %w", err))
	}

	result, err := newGenerationResult(response.GetGeneration())
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("read generation: %w", err))
	}

	span.SetAttributes(attribute.String("generation.status", result.Status))

	return result, nil
}
