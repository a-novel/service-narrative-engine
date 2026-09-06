package lib

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

// GenAIGenerationWatch adapts a client's GenerationWatch method to the core gateway.
type GenAIGenerationWatch func(
	ctx context.Context,
	request *servicegenai.GenerationWatchRequest,
	options ...grpc.CallOption,
) (grpc.ServerStreamingClient[servicegenai.GenerationWatchResponse], error)

// GenerationWatchGatewayRequest identifies one owner-scoped generation stream.
type GenerationWatchGatewayRequest struct {
	// ID identifies the generation in service-genai.
	ID uuid.UUID
	// OwnerID identifies the owner used to scope the stream.
	OwnerID uuid.UUID
}

// GenerationWatchGatewayStream yields domain snapshots and releases its RPC when closed.
type GenerationWatchGatewayStream interface {
	// Recv waits for and converts the next generation snapshot.
	Recv() (*Generation, error)
	// Close releases the stream and cancels any blocked receive.
	Close()
}

// Exec opens a cancellable owner-scoped generation stream.
func (gateway GenAIGenerationWatch) Exec(
	ctx context.Context,
	request *GenerationWatchGatewayRequest,
) (GenerationWatchGatewayStream, error) {
	ctx, span := otel.Tracer().Start(ctx, "lib.GenAIGenerationWatch")
	defer span.End()

	streamCtx, cancel := context.WithCancel(ctx)

	stream, err := gateway(streamCtx, &servicegenai.GenerationWatchRequest{
		Id:      request.ID.String(),
		OwnerId: request.OwnerID.String(),
	})
	if err != nil {
		cancel()

		return nil, otel.ReportError(span, normalizeGenAIError(err, codes.NotFound, ErrGenerationNotFound))
	}

	if stream == nil {
		cancel()

		return nil, otel.ReportError(span, ErrGenerationResponseInvalid)
	}

	return otel.ReportSuccess[GenerationWatchGatewayStream](
		span, &genAIGenerationWatchStream{stream: stream, cancel: cancel},
	), nil
}

type genAIGenerationWatchStream struct {
	stream grpc.ServerStreamingClient[servicegenai.GenerationWatchResponse]
	cancel context.CancelFunc
}

func (stream *genAIGenerationWatchStream) Recv() (*Generation, error) {
	response, err := stream.stream.Recv()
	if err != nil {
		stream.Close()

		return nil, normalizeGenAIError(err, codes.NotFound, ErrGenerationNotFound)
	}

	generation, err := mapGenAIGeneration(response.GetGeneration())
	if err != nil {
		stream.Close()

		return nil, err
	}

	return generation, nil
}

func (stream *genAIGenerationWatchStream) Close() {
	stream.cancel()
}
