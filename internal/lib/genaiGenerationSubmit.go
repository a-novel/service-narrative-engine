package lib

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

// GenAIGenerationSubmit adapts a client's GenerationSubmit method to the core gateway.
type GenAIGenerationSubmit func(
	ctx context.Context,
	request *servicegenai.GenerationSubmitRequest,
	options ...grpc.CallOption,
) (*servicegenai.GenerationSubmitResponse, error)

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

// Exec submits and converts one owner-scoped generation.
func (gateway GenAIGenerationSubmit) Exec(
	ctx context.Context,
	request *GenerationSubmitGatewayRequest,
) (*GenerationSubmitGatewayResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "lib.GenAIGenerationSubmit")
	defer span.End()

	response, err := gateway(ctx, &servicegenai.GenerationSubmitRequest{
		OwnerId:        request.OwnerID.String(),
		Purpose:        request.Purpose,
		IdempotencyKey: request.IdempotencyKey,
		Request:        request.Request,
		MaxAttempts:    request.MaxAttempts,
	})
	if err != nil {
		return nil, otel.ReportError(span, normalizeGenAIError(err, codes.AlreadyExists, ErrGenerationConflict))
	}

	generation, err := mapGenAIGeneration(response.GetGeneration())
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, &GenerationSubmitGatewayResult{
		Generation: generation,
		Created:    response.GetCreated(),
	}), nil
}
