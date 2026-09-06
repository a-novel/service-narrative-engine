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

// GenAIGenerationSubmitClient is the service-genai operation used by [GenAIGenerationSubmit].
type GenAIGenerationSubmitClient interface {
	// GenerationSubmit records one owner-scoped service-genai generation.
	GenerationSubmit(
		ctx context.Context,
		request *servicegenai.GenerationSubmitRequest,
		options ...grpc.CallOption,
	) (*servicegenai.GenerationSubmitResponse, error)
}

// GenerationSubmitGatewayRequest contains one service-genai submission.
type GenerationSubmitGatewayRequest struct {
	// OwnerID identifies the owner charged for the generation.
	OwnerID uuid.UUID
	// Purpose identifies the workflow submitting the generation.
	Purpose string
	// IdempotencyKey identifies one retry sequence.
	IdempotencyKey string
	// Request contains the provider request payload.
	Request []byte
	// MaxAttempts limits provider execution attempts.
	MaxAttempts int32
}

// GenerationSubmitGatewayResult reports the submitted generation and replay state.
type GenerationSubmitGatewayResult struct {
	// Generation is the created or replayed owner-scoped work.
	Generation *Generation
	// Created distinguishes new work from an idempotent replay.
	Created bool
}

// GenAIGenerationSubmit adapts the service-genai submit RPC to a domain result.
type GenAIGenerationSubmit struct {
	client GenAIGenerationSubmitClient
}

// NewGenAIGenerationSubmit creates a service-genai generation-submit adapter.
func NewGenAIGenerationSubmit(client GenAIGenerationSubmitClient) *GenAIGenerationSubmit {
	return &GenAIGenerationSubmit{client: client}
}

// Exec submits and converts one owner-scoped generation.
func (gateway *GenAIGenerationSubmit) Exec(
	ctx context.Context,
	request *GenerationSubmitGatewayRequest,
) (*GenerationSubmitGatewayResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "lib.GenAIGenerationSubmit")
	defer span.End()

	response, err := gateway.client.GenerationSubmit(ctx, &servicegenai.GenerationSubmitRequest{
		OwnerId:        request.OwnerID.String(),
		Purpose:        request.Purpose,
		IdempotencyKey: request.IdempotencyKey,
		Request:        request.Request,
		MaxAttempts:    request.MaxAttempts,
	})

	err = normalizeGenAIError("submit generation", err, map[codes.Code]error{
		codes.AlreadyExists: ErrGenerationConflict,
	})
	if err != nil {
		return nil, err
	}

	if response == nil {
		return nil, fmt.Errorf("%w: missing submit response", ErrGenerationResponseInvalid)
	}

	generation, err := mapGenAIGeneration(response.GetGeneration())
	if err != nil {
		return nil, err
	}

	return &GenerationSubmitGatewayResult{
		Generation: generation,
		Created:    response.GetCreated(),
	}, nil
}
