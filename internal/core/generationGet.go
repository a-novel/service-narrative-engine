package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

// GenerationGetServiceGenai reads a genai generation of the caller.
type GenerationGetServiceGenai interface {
	GenerationGet(
		ctx context.Context, req *servicegenai.GenerationGetRequest, opts ...grpc.CallOption,
	) (*servicegenai.GenerationGetResponse, error)
}

// GenerationGetRequest names the genai generation to read, and its caller.
type GenerationGetRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`
}

// GenerationGet reads one call of a generation through genai, as the caller. Once the call succeeded, it
// returns the model's output after the output passes the full output schema of its kind.
type GenerationGet struct {
	genai GenerationGetServiceGenai
}

func NewGenerationGet(genai GenerationGetServiceGenai) *GenerationGet {
	return &GenerationGet{genai: genai}
}

func (service *GenerationGet) Exec(ctx context.Context, request *GenerationGetRequest) (*GenerationResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationGet")
	defer span.End()

	span.SetAttributes(
		attribute.String("generation.id", request.ID.String()),
		attribute.String("generation.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	genaiCtx, cancel := context.WithTimeout(ctx, genaiTimeout)
	defer cancel()

	response, err := service.genai.GenerationGet(genaiCtx, &servicegenai.GenerationGetRequest{
		Id: request.ID.String(), OwnerId: request.OwnerID.String(),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("genai get: %w", genaiError(err, ErrGenerationNotFound)))
	}

	result, err := newGenerationResult(response.GetGeneration())
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("read generation: %w", err))
	}

	span.SetAttributes(attribute.String("generation.status", result.Status))

	return result, nil
}
