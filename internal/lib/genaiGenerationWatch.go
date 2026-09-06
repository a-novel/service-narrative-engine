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

// GenAIGenerationWatchClient is the service-genai operation used by [GenAIGenerationWatch].
type GenAIGenerationWatchClient interface {
	// GenerationWatch opens one owner-scoped service-genai generation stream.
	GenerationWatch(
		ctx context.Context,
		request *servicegenai.GenerationWatchRequest,
		options ...grpc.CallOption,
	) (grpc.ServerStreamingClient[servicegenai.GenerationWatchResponse], error)
}

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

// GenAIGenerationWatch adapts the service-genai watch RPC to a domain stream.
type GenAIGenerationWatch struct {
	client GenAIGenerationWatchClient
}

// NewGenAIGenerationWatch creates a service-genai generation-watch adapter.
func NewGenAIGenerationWatch(client GenAIGenerationWatchClient) *GenAIGenerationWatch {
	return &GenAIGenerationWatch{client: client}
}

// Exec opens a cancellable owner-scoped generation stream.
func (gateway *GenAIGenerationWatch) Exec(
	ctx context.Context,
	request *GenerationWatchGatewayRequest,
) (GenerationWatchGatewayStream, error) {
	ctx, span := otel.Tracer().Start(ctx, "lib.GenAIGenerationWatch")
	defer span.End()

	streamCtx, cancel := context.WithCancel(ctx)
	stream, err := gateway.client.GenerationWatch(streamCtx, &servicegenai.GenerationWatchRequest{
		Id:      request.ID.String(),
		OwnerId: request.OwnerID.String(),
	})

	err = normalizeGenAIError("watch generation", err, map[codes.Code]error{
		codes.NotFound: ErrGenerationNotFound,
	})
	if err != nil {
		cancel()

		return nil, err
	}

	if stream == nil {
		cancel()

		return nil, fmt.Errorf("%w: missing watch stream", ErrGenerationResponseInvalid)
	}

	return &genAIGenerationWatchStream{stream: stream, cancel: cancel}, nil
}

type genAIGenerationWatchStream struct {
	stream grpc.ServerStreamingClient[servicegenai.GenerationWatchResponse]
	cancel context.CancelFunc
}

func (stream *genAIGenerationWatchStream) Recv() (*Generation, error) {
	response, err := stream.stream.Recv()

	err = normalizeGenAIError("receive generation", err, map[codes.Code]error{
		codes.NotFound: ErrGenerationNotFound,
	})
	if err != nil {
		stream.Close()

		return nil, err
	}

	if response == nil {
		stream.Close()

		return nil, fmt.Errorf("%w: missing watch response", ErrGenerationResponseInvalid)
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
