package lib

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

// GenAIGenerationGet adapts a client's GenerationGet method to the core gateway.
type GenAIGenerationGet func(
	ctx context.Context,
	request *servicegenai.GenerationGetRequest,
	options ...grpc.CallOption,
) (*servicegenai.GenerationGetResponse, error)

// GenerationGetGatewayRequest identifies one owner-scoped generation.
type GenerationGetGatewayRequest struct {
	// ID identifies the generation in service-genai.
	ID uuid.UUID
	// OwnerID identifies the owner used to scope the lookup.
	OwnerID uuid.UUID
}

// Exec retrieves and converts one owner-scoped generation.
func (gateway GenAIGenerationGet) Exec(
	ctx context.Context,
	request *GenerationGetGatewayRequest,
) (*Generation, error) {
	ctx, span := otel.Tracer().Start(ctx, "lib.GenAIGenerationGet")
	defer span.End()

	response, err := gateway(ctx, &servicegenai.GenerationGetRequest{
		Id:      request.ID.String(),
		OwnerId: request.OwnerID.String(),
	})
	if err != nil {
		return nil, otel.ReportError(span, normalizeGenAIError(err, codes.NotFound, ErrGenerationNotFound))
	}

	generation, err := mapGenAIGeneration(response.GetGeneration())
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, generation), nil
}
