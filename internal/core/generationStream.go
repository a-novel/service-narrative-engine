package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

// GenerationStreamServiceGenai streams a genai generation of the caller.
type GenerationStreamServiceGenai interface {
	GenerationStream(
		ctx context.Context, req *servicegenai.GenerationStreamRequest, opts ...grpc.CallOption,
	) (grpc.ServerStreamingClient[servicegenai.GenerationStreamResponse], error)
}

// GenerationStreamServiceRead reads the call once its stream ends.
type GenerationStreamServiceRead interface {
	Exec(ctx context.Context, request *GenerationGetRequest) (*GenerationResult, error)
}

// GenerationStreamRequest names the call to stream, its caller, and where the stream resumes.
type GenerationStreamRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`
	// After is the sequence number of the last event the caller received. Nil streams from the first
	// event.
	After *int64 `validate:"omitnil,gte=0"`
}

// GenerationEvent is one event of a streamed call: text the model wrote, or the call's result.
type GenerationEvent struct {
	// Sequence orders the call's events. A stream resumes after it.
	Sequence int64
	// Delta is text the model appended to its output.
	Delta string
	// Result is the settled call, as a read returns it. It ends the stream.
	Result *GenerationResult
}

// providerEvent is the part of an OpenAI stream event a stream reads.
type providerEvent struct {
	Type  string `json:"type"`
	Delta string `json:"delta"`
}

// providerDeltaEvent carries text the model appended to its output.
const providerDeltaEvent = "response.output_text.delta"

// providerEndEvents end a call on the provider's side. A cancelled call has none: its stream just ends.
var providerEndEvents = map[string]bool{
	"response.completed":  true,
	"response.failed":     true,
	"response.incomplete": true,
	"error":               true,
}

// GenerationStream forwards the text of a call submitted with streaming as the model writes it, then
// the call's result. It reads the call through genai, as the caller.
//
// Before the stream opens, it returns [ErrGenerationNotStreamed] for a call it cannot stream, and
// [ErrStreamEnded] when the caller already received every event. A stream cut before its result
// resumes after the last event the caller received.
type GenerationStream struct {
	genai GenerationStreamServiceGenai
	read  GenerationStreamServiceRead
}

func NewGenerationStream(genai GenerationStreamServiceGenai, read GenerationStreamServiceRead) *GenerationStream {
	return &GenerationStream{genai: genai, read: read}
}

func (service *GenerationStream) Exec(
	ctx context.Context, request *GenerationStreamRequest,
) (iter.Seq2[*GenerationEvent, error], error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationStream")
	defer span.End()

	span.SetAttributes(
		attribute.String("generation.id", request.ID.String()),
		attribute.String("generation.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	stream, err := service.genai.GenerationStream(ctx, &servicegenai.GenerationStreamRequest{
		Id: request.ID.String(), OwnerId: request.OwnerID.String(), StartingAfter: request.After,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("genai stream: %w", streamError(err)))
	}

	// genai refuses a stream on its first message, so the stream opens once that message is in.
	first, err := stream.Recv()
	if errors.Is(err, io.EOF) {
		return service.ended(ctx, request)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("genai stream: %w", streamError(err)))
	}

	return func(yield func(*GenerationEvent, error) bool) {
		service.forward(ctx, request, stream, first, yield)
	}, nil
}

// ended answers a stream that holds no event after the caller's position, as a cancelled call's does.
// A settled call's stream holds its result, unless the caller resumes after the last event.
func (service *GenerationStream) ended(
	ctx context.Context, request *GenerationStreamRequest,
) (iter.Seq2[*GenerationEvent, error], error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationStream(ended)")
	defer span.End()

	result, err := service.read.Exec(ctx, &GenerationGetRequest{OwnerID: request.OwnerID, ID: request.ID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("read generation: %w", err))
	}

	switch {
	case !result.settled():
		return nil, ErrGenerationNotStreamed
	case request.After != nil:
		return nil, ErrStreamEnded
	}

	return func(yield func(*GenerationEvent, error) bool) {
		yield(&GenerationEvent{Result: result}, nil)
	}, nil
}

// forward yields the text events of the stream from first on, then the call's result once the provider
// ends it. It runs after Exec returns, so it takes a span of its own.
func (service *GenerationStream) forward(
	ctx context.Context,
	request *GenerationStreamRequest,
	stream grpc.ServerStreamingClient[servicegenai.GenerationStreamResponse],
	message *servicegenai.GenerationStreamResponse,
	yield func(*GenerationEvent, error) bool,
) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationStream(events)")
	defer span.End()

	var (
		sequence int64
		err      error
	)

	for ; !errors.Is(err, io.EOF); message, err = stream.Recv() {
		// A caller that left ended the stream: no fault.
		if err != nil && ctx.Err() != nil {
			return
		}

		if err != nil {
			yield(nil, otel.ReportError(span, fmt.Errorf("genai stream: %w", streamError(err))))

			return
		}

		sequence = message.GetSequenceNumber()

		var event providerEvent

		err = json.Unmarshal(message.GetEvent(), &event)
		if err != nil {
			yield(nil, otel.ReportError(span, fmt.Errorf("decode provider event: %w", err)))

			return
		}

		if providerEndEvents[event.Type] {
			break
		}

		if event.Type == providerDeltaEvent && !yield(&GenerationEvent{Sequence: sequence, Delta: event.Delta}, nil) {
			return
		}
	}

	// The read settles the call, so the result matches what a read returns.
	result, err := service.read.Exec(ctx, &GenerationGetRequest{OwnerID: request.OwnerID, ID: request.ID})
	if err != nil {
		yield(nil, otel.ReportError(span, fmt.Errorf("read generation: %w", err)))

		return
	}

	if result.settled() {
		yield(&GenerationEvent{Sequence: sequence, Result: result}, nil)
	}
}

// streamError joins a sentinel to an error genai returned on a stream: [ErrGenerationNotStreamed] when
// genai cannot stream the generation, and the sentinels of [genaiError] otherwise.
func streamError(err error) error {
	if status.Code(err) == codes.FailedPrecondition {
		return errors.Join(err, ErrGenerationNotStreamed)
	}

	return genaiError(err, ErrGenerationNotFound)
}
