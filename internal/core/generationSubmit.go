package core

import (
	"cmp"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
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

// GenerationSubmitRequest asks for a generation on a project of the caller. Each kind takes its own fields.
type GenerationSubmitRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// Roles are the caller's roles, which set the caps genai counts the generation against.
	Roles []string

	// Kind is the kind of the generation, one of [GenerationKinds].
	Kind string `validate:"generationkind"`

	// Targets lists what a suggest generation writes, from 1 to 20, each once.
	Targets []*GenerationTarget `validate:"excluded_unless=Kind suggest,max=20,dive,required"`
	// Cell names the cell a consolidate generation recaps, unless BeatID names a beat.
	Cell *GenerationCell `validate:"excluded_unless=Kind consolidate"`
	// BeatID names the beat a consolidate generation recaps.
	BeatID uuid.UUID `validate:"excluded_unless=Kind consolidate"`
	// Scenes narrows a check generation to a range of scene positions. Nil checks every scene.
	Scenes *Range `validate:"excluded_unless=Kind check"`

	// Instruction is what the author asks of the generation.
	Instruction string `validate:"max=4000,excluded_if=Kind consolidate"`
	// Exclude lists the layers the generation does not read. A target still shows its own cell.
	Exclude []uuid.UUID `validate:"max=1000,unique,excluded_if=Kind consolidate"`
	// Overrides carries unsaved content the generation reads in place of the stored one.
	Overrides *GenerationOverrides `validate:"excluded_if=Kind consolidate"`

	// Variant asks genai for another result of the same calls.
	Variant uint32
	// Stream creates each call in streaming mode, so [GenerationStream] forwards its text as the model
	// writes it. A streamed call and an unstreamed one are two genai generations.
	Stream bool
}

// GenerationSubmit plans a generation and submits each of its calls to genai, as the caller, under the caps
// of the caller's roles. A call genai refuses for a cap waits, and so do the calls after it. Resending the
// same request admits the calls that wait: genai replays the calls it admitted and those that succeeded,
// and runs again those that failed or were cancelled, a call whose output reads as failed included. A
// generation whose every call waits returns a [GenerationQuotaError]. The read of the project and the plan
// wait for a slot of the gate, which they share with every other read of a whole project, and free it before
// the first call to genai.
type GenerationSubmit struct {
	snapshot GenerationSubmitServiceSnapshot
	plan     GenerationSubmitServicePlan
	genai    GenerationSubmitServiceGenai
	gate     *ProjectReadGate
	caps     *GenerationCapsPolicy
}

func NewGenerationSubmit(
	snapshot GenerationSubmitServiceSnapshot,
	plan GenerationSubmitServicePlan,
	genai GenerationSubmitServiceGenai,
	gate *ProjectReadGate,
	caps *GenerationCapsPolicy,
) *GenerationSubmit {
	return &GenerationSubmit{snapshot: snapshot, plan: plan, genai: genai, gate: gate, caps: caps}
}

func (service *GenerationSubmit) Exec(ctx context.Context, request *GenerationSubmitRequest) (*Generation, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationSubmit")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("generation.kind", request.Kind),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	caps, err := service.caps.Resolve(request.Roles)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	var plan *ContextPlan

	err = service.gate.Run(ctx, func(ctx context.Context) error {
		snapshot, err := service.snapshot.Exec(ctx, &ProjectSnapshotReadRequest{
			OwnerID: request.OwnerID, ProjectID: request.ProjectID,
		})
		if err != nil {
			return fmt.Errorf("read project: %w", err)
		}

		plan, err = service.plan.Exec(ctx, &GenerationPlanRequest{
			Snapshot: snapshot, Kind: request.Kind, Targets: request.Targets, Cell: request.Cell, BeatID: request.BeatID,
			Scenes: request.Scenes, Instruction: request.Instruction, Exclude: request.Exclude, Overrides: request.Overrides,
		})
		if err != nil {
			return fmt.Errorf("plan generation: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	generation := &Generation{Kind: plan.Kind, Calls: make([]*GenerationCall, len(plan.Calls))}
	genaiCaps := caps.genai()

	var refusal *GenerationQuotaError

	for position, call := range plan.Calls {
		submitted := &GenerationCall{Context: call}
		generation.Calls[position] = submitted

		// A cap that refused a call refuses the calls after it as well, until a call settles.
		if refusal == nil {
			submitted.Result, refusal, err = service.submit(ctx, request, genaiCaps, plan.Kind, call, position)
			if err != nil {
				return nil, otel.ReportError(span, fmt.Errorf("submit call: %w", err))
			}
		}

		if refusal != nil {
			submitted.Waiting = refusal.Cap
		}
	}

	if refusal != nil && lo.EveryBy(generation.Calls, func(call *GenerationCall) bool { return call.Waiting != "" }) {
		return nil, otel.ReportError(span, refusal)
	}

	return generation, nil
}

// submit submits the call at a position of the generation to genai. It returns the genai generation that
// runs the call, or genai's refusal for a cap.
//
// genai replays a call it already ran, so a replay whose output reads as failed would fail again on every
// resend. Such a call runs again at the next variant instead. Variants already run replay at no cost, so the
// walk stops at the first variant genai creates, still runs, or holds a valid output for.
func (service *GenerationSubmit) submit(
	ctx context.Context, request *GenerationSubmitRequest, caps *servicegenai.GenerationCaps, kind string,
	call *ContextCall, position int,
) (*GenerationResult, *GenerationQuotaError, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationSubmit(call)")
	defer span.End()

	span.SetAttributes(attribute.Int("generation.call", position))

	payload, err := newPayload(kind, request.ProjectID.String(), call.Prompt)
	if err != nil {
		return nil, nil, otel.ReportError(span, err)
	}

	genaiRequest := &servicegenai.GenerationSubmitRequest{
		OwnerId:  request.OwnerID.String(),
		Purpose:  generationPurpose,
		Provider: generationProvider,
		Payload:  payload,
		Caps:     caps,
		Variant:  request.Variant,
		Stream:   request.Stream,
	}

	for ; ; genaiRequest.Variant++ {
		genaiCtx, cancel := context.WithTimeout(ctx, genaiTimeout)
		response, err := service.genai.GenerationSubmit(genaiCtx, genaiRequest)

		cancel()

		if status.Code(err) == codes.ResourceExhausted {
			span.SetAttributes(attribute.Bool("generation.waiting", true))

			return nil, newGenerationQuotaError(err), nil
		}

		if err != nil {
			return nil, nil, otel.ReportError(span, fmt.Errorf("genai submit: %w", genaiError(err, nil)))
		}

		result, err := newGenerationResult(response.GetGeneration())
		if err != nil {
			return nil, nil, otel.ReportError(span, fmt.Errorf("read generation: %w", err))
		}

		// genai holds a replayed call as succeeded once the provider finished it, whatever its output.
		rejected := !response.GetCreated() && result.Status == GenerationStatusFailed &&
			response.GetGeneration().GetStatus() == servicegenai.GenerationStatusSucceeded
		if !rejected {
			span.SetAttributes(attribute.Int64("generation.variant", int64(genaiRequest.GetVariant())))

			return result, nil, nil
		}
	}
}

// newGenerationQuotaError reads genai's refusal of a call for a cap. Its QuotaFailure names the limit that
// refused, and its RetryInfo, when a window refused, says when the window frees usage. A refusal without a
// QuotaFailure names the cap in its message alone.
func newGenerationQuotaError(err error) *GenerationQuotaError {
	refusal := status.Convert(err)
	quota := &GenerationQuotaError{Cap: refusal.Message()}

	for _, detail := range refusal.Details() {
		switch detail := detail.(type) {
		case *errdetails.QuotaFailure:
			// genai describes the limit in the one violation it sends.
			quota.Cap = cmp.Or(lo.FirstOrEmpty(detail.GetViolations()).GetDescription(), quota.Cap)
		case *errdetails.RetryInfo:
			quota.RetryAfter = detail.GetRetryDelay().AsDuration()
		}
	}

	return quota
}
