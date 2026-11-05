package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

// GenerationUsageServiceGenai reads what a user consumed on genai.
type GenerationUsageServiceGenai interface {
	UsageSummary(
		ctx context.Context, req *servicegenai.UsageSummaryRequest, opts ...grpc.CallOption,
	) (*servicegenai.UsageSummaryResponse, error)
}

// GenerationUsageRequest names the caller whose usage to read.
type GenerationUsageRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	// Roles are the caller's roles, which set their caps.
	Roles []string
}

// GenerationUsageSummary is a user's caps, and what they consumed under each.
type GenerationUsageSummary struct {
	// Caps are the caps genai counts the user's generations against. Empty caps limit nothing.
	Caps *GenerationCaps
	// InFlight is how many of the user's calls genai counts against the concurrency: pending, running, or
	// failed with an unknown outcome within the last 15 minutes.
	InFlight int64
	// Windows holds the tokens the user's calls consumed over each window of the caps, in their order.
	Windows []*GenerationUsageWindow
}

// GenerationUsageWindow is what a user's calls consumed over one rolling window.
type GenerationUsageWindow struct {
	// Duration is how far back the window reaches.
	Duration time.Duration

	// InputTokens is the input tokens consumed in the window, cached ones included.
	InputTokens int64
	// OutputTokens is the output tokens consumed in the window, reasoning ones included.
	OutputTokens int64
}

// GenerationUsage reads the caps of the caller's roles, and what the caller consumed under them through
// genai, so a client shows the room left before a submit waits.
type GenerationUsage struct {
	genai GenerationUsageServiceGenai
	caps  *GenerationCapsPolicy
}

func NewGenerationUsage(genai GenerationUsageServiceGenai, caps *GenerationCapsPolicy) *GenerationUsage {
	return &GenerationUsage{genai: genai, caps: caps}
}

func (service *GenerationUsage) Exec(
	ctx context.Context, request *GenerationUsageRequest,
) (*GenerationUsageSummary, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationUsage")
	defer span.End()

	span.SetAttributes(attribute.String("generation.owner_id", request.OwnerID.String()))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	caps, err := service.caps.Resolve(request.Roles)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	windows := make([]*durationpb.Duration, len(caps.Windows))
	for i, window := range caps.Windows {
		windows[i] = durationpb.New(window.Duration)
	}

	genaiCtx, cancel := context.WithTimeout(ctx, genaiTimeout)
	defer cancel()

	response, err := service.genai.UsageSummary(genaiCtx, &servicegenai.UsageSummaryRequest{
		OwnerId: request.OwnerID.String(), Purpose: generationPurpose, Windows: windows,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("genai usage summary: %w", genaiError(err, nil)))
	}

	summary := &GenerationUsageSummary{
		Caps:     caps,
		InFlight: response.GetLive(),
		Windows:  make([]*GenerationUsageWindow, len(response.GetWindows())),
	}

	for i, window := range response.GetWindows() {
		summary.Windows[i] = &GenerationUsageWindow{
			Duration:     window.GetDuration().AsDuration(),
			InputTokens:  window.GetInputTokens(),
			OutputTokens: window.GetOutputTokens(),
		}
	}

	return summary, nil
}
