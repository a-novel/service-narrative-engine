package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"
)

// GenerationSubmitServiceSnapshot reads the project the generation reads, once the caller proves they own
// it.
type GenerationSubmitServiceSnapshot interface {
	Exec(ctx context.Context, request *ProjectSnapshotReadRequest) (*ProjectSnapshot, error)
}

// GenerationSubmitServicePlan plans the calls of the generation and renders their prompts.
type GenerationSubmitServicePlan interface {
	Exec(ctx context.Context, request *GenerationPlanRequest) (*ContextPlan, error)
}

// GenerationSubmitServiceGenai submits a call to genai, or replays the one it already runs.
type GenerationSubmitServiceGenai interface {
	GenerationSubmit(
		ctx context.Context, req *servicegenai.GenerationSubmitRequest, opts ...grpc.CallOption,
	) (*servicegenai.GenerationSubmitResponse, error)
}

// GenerationSubmitRequest asks for a suggest generation on a project of the caller.
type GenerationSubmitRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// Admin runs the generation on genai's admin tier, which no cap limits.
	Admin bool

	// Targets lists what the generation writes, from 1 to 20, each once.
	Targets []*GenerationTarget `validate:"required,min=1,max=20,dive,required"`
	// Instruction is what the author asks of the generation.
	Instruction string `validate:"max=4000"`
	// Exclude lists the layers the generation does not read. A target still shows its own cell.
	Exclude []uuid.UUID `validate:"max=1000,unique"`
	// Overrides carries unsaved content the generation reads in place of the stored one.
	Overrides *GenerationOverrides

	// Variant asks genai for another result of the same calls.
	Variant uint32
}

// GenerationSubmit plans a suggest generation and submits each of its calls to genai, as the caller. A call
// genai refuses for a cap waits, and so do the calls after it. Resending the same request admits the
// calls that wait: genai replays the calls it admitted and those that succeeded, and runs again those that
// failed or were cancelled. Every call that waits returns [ErrGenerationQuota].
type GenerationSubmit struct {
	snapshot GenerationSubmitServiceSnapshot
	plan     GenerationSubmitServicePlan
	genai    GenerationSubmitServiceGenai
}

func NewGenerationSubmit(
	snapshot GenerationSubmitServiceSnapshot, plan GenerationSubmitServicePlan, genai GenerationSubmitServiceGenai,
) *GenerationSubmit {
	return &GenerationSubmit{snapshot: snapshot, plan: plan, genai: genai}
}

func (service *GenerationSubmit) Exec(ctx context.Context, request *GenerationSubmitRequest) (*Generation, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationSubmit")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	snapshot, err := service.snapshot.Exec(ctx, &ProjectSnapshotReadRequest{
		OwnerID: request.OwnerID, ProjectID: request.ProjectID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("read project: %w", err))
	}

	plan, err := service.plan.Exec(ctx, &GenerationPlanRequest{
		Snapshot: snapshot, Kind: GenerationKindSuggest, Targets: request.Targets,
		Instruction: request.Instruction, Exclude: request.Exclude, Overrides: request.Overrides,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("plan generation: %w", err))
	}

	generation := &Generation{Kind: plan.Kind}

	var waiting string

	for _, call := range plan.Calls {
		submitted := &GenerationCall{Context: call, Waiting: waiting}
		generation.Calls = append(generation.Calls, submitted)

		// A cap that refused a call refuses the calls after it as well, until a call settles.
		if waiting != "" {
			continue
		}

		submitted.Result, submitted.Waiting, err = service.submit(ctx, request, plan.Kind, call, len(generation.Calls)-1)
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("submit call: %w", err))
		}

		waiting = submitted.Waiting
	}

	span.SetAttributes(attribute.Int("generation.calls", len(generation.Calls)))

	if lo.EveryBy(generation.Calls, func(call *GenerationCall) bool { return call.Waiting != "" }) {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %s", ErrGenerationQuota, waiting))
	}

	return generation, nil
}

// submit submits the call at a position of the generation to genai. It returns the genai generation that
// runs the call, or the cap genai refused it for.
func (service *GenerationSubmit) submit(
	ctx context.Context, request *GenerationSubmitRequest, kind string, call *ContextCall, position int,
) (*GenerationResult, string, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationSubmit(call)")
	defer span.End()

	span.SetAttributes(attribute.Int("generation.call", position))

	payload, err := newPayload(kind, request.ProjectID.String(), call.Prompt)
	if err != nil {
		return nil, "", otel.ReportError(span, err)
	}

	genaiCtx, cancel := context.WithTimeout(ctx, genaiTimeout)
	defer cancel()

	response, err := service.genai.GenerationSubmit(genaiCtx, &servicegenai.GenerationSubmitRequest{
		OwnerId:  request.OwnerID.String(),
		Purpose:  generationPurpose,
		Provider: generationProvider,
		Payload:  payload,
		UserTier: lo.Ternary(request.Admin, genaiTierAdmin, genaiTierDefault),
		Variant:  request.Variant,
	})
	if status.Code(err) == codes.ResourceExhausted {
		span.SetAttributes(attribute.Bool("generation.waiting", true))

		return nil, status.Convert(err).Message(), nil
	}

	if err != nil {
		return nil, "", otel.ReportError(span, fmt.Errorf("genai submit: %w", genaiError(err, nil)))
	}

	result, err := newGenerationResult(response.GetGeneration())
	if err != nil {
		return nil, "", otel.ReportError(span, fmt.Errorf("read generation: %w", err))
	}

	return result, "", nil
}
