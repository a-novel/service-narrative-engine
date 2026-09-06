package lib

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

// GenAIGenerationGetClient is the service-genai operation used by [GenAIGenerationGet].
type GenAIGenerationGetClient interface {
	// GenerationGet retrieves one owner-scoped service-genai generation.
	GenerationGet(
		ctx context.Context,
		request *servicegenai.GenerationGetRequest,
		options ...grpc.CallOption,
	) (*servicegenai.GenerationGetResponse, error)
}

// GenerationGetGatewayRequest identifies one owner-scoped generation.
type GenerationGetGatewayRequest struct {
	// ID identifies the generation in service-genai.
	ID uuid.UUID
	// OwnerID identifies the owner used to scope the lookup.
	OwnerID uuid.UUID
}

// GenAIGenerationGet adapts the service-genai get RPC to a domain result.
type GenAIGenerationGet struct {
	client GenAIGenerationGetClient
}

// NewGenAIGenerationGet creates a service-genai generation-get adapter.
func NewGenAIGenerationGet(client GenAIGenerationGetClient) *GenAIGenerationGet {
	return &GenAIGenerationGet{client: client}
}

// Exec retrieves and converts one owner-scoped generation.
func (gateway *GenAIGenerationGet) Exec(
	ctx context.Context,
	request *GenerationGetGatewayRequest,
) (*Generation, error) {
	ctx, span := otel.Tracer().Start(ctx, "lib.GenAIGenerationGet")
	defer span.End()

	response, err := gateway.client.GenerationGet(ctx, &servicegenai.GenerationGetRequest{
		Id:      request.ID.String(),
		OwnerId: request.OwnerID.String(),
	})

	err = normalizeGenAIError("get generation", err, map[codes.Code]error{
		codes.NotFound: ErrGenerationNotFound,
	})
	if err != nil {
		return nil, err
	}

	if response == nil {
		return nil, fmt.Errorf("%w: missing get response", ErrGenerationResponseInvalid)
	}

	return mapGenAIGeneration(response.GetGeneration())
}
