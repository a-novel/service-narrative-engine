package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// IdeaSelectDao retrieves owner-scoped Ideas for generation and content saves.
type IdeaSelectDao interface {
	Exec(ctx context.Context, request *dao.IdeaSelectRequest) (*dao.Idea, error)
}

// GenerationSubmitRequest identifies the Idea, immutable step, and caller retry.
type GenerationSubmitRequest struct {
	Actor           Actor     `validate:"required"`
	IdeaID          uuid.UUID `validate:"required"`
	EngineVersionID uuid.UUID `validate:"required"`
	StepKey         string    `validate:"required,notblank,max=256"`
	IdempotencyKey  string    `validate:"required,notblank,max=256"`
}

// GenerationSubmit assembles and submits one self-contained fixture-engine request.
type GenerationSubmit struct {
	ideaDao          IdeaSelectDao
	engineVersionDao EngineVersionSelectDao
	genai            servicegenai.Client
}

// NewGenerationSubmit creates the narrative generation submit service.
func NewGenerationSubmit(
	ideaDao IdeaSelectDao,
	engineVersionDao EngineVersionSelectDao,
	genai servicegenai.Client,
) *GenerationSubmit {
	return &GenerationSubmit{
		ideaDao:          ideaDao,
		engineVersionDao: engineVersionDao,
		genai:            genai,
	}
}

// Exec submits one priced generation or attaches to the caller's existing work.
func (service *GenerationSubmit) Exec(
	ctx context.Context,
	request *GenerationSubmitRequest,
) (*GenerationSubmitResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationSubmit")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(
		attribute.String("idea.id", request.IdeaID.String()),
		attribute.String("engine_version.id", request.EngineVersionID.String()),
		attribute.String("engine.step_key", request.StepKey),
		attribute.String("generation.owner_id", request.Actor.UserID.String()),
	)

	idea, err := service.ideaDao.Exec(ctx, &dao.IdeaSelectRequest{
		ID:      request.IdeaID,
		OwnerID: request.Actor.UserID,
	})
	if err != nil {
		if errors.Is(err, dao.ErrIdeaSelectNotFound) {
			err = errors.Join(err, ErrIdeaNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("select Idea: %w", err))
	}

	engineVersion, err := service.engineVersionDao.Exec(ctx, &dao.EngineVersionSelectRequest{
		ID: request.EngineVersionID,
	})
	if err != nil {
		if errors.Is(err, dao.ErrEngineVersionSelectNotFound) {
			err = errors.Join(err, ErrEngineVersionNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("select Engine Version: %w", err))
	}

	step, err := selectEngineStep(engineVersion.Definition, request.StepKey)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	payload, err := buildGenerationPayload(idea, engineVersion.ID, step)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("build generation payload: %w", err))
	}

	idempotencyKey, err := deriveGenerationIdempotencyKey(
		request.IdempotencyKey,
		request.IdeaID,
		request.EngineVersionID,
		request.StepKey,
	)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	response, err := service.genai.GenerationSubmit(ctx, &servicegenai.GenerationSubmitRequest{
		OwnerId:        request.Actor.UserID.String(),
		Purpose:        GenerationPurposeStudio,
		IdempotencyKey: idempotencyKey,
		Request:        payload,
		MaxAttempts:    generationMaxAttempts,
	})
	if status.Code(err) == codes.AlreadyExists {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %w", ErrGenerationConflict, err))
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("submit generation: %w", err))
	}

	if response == nil {
		return nil, otel.ReportError(span, fmt.Errorf("%w: missing submit response", ErrGenerationResponseInvalid))
	}

	generation, err := mapGeneration(
		ctx,
		service.engineVersionDao,
		response.GetGeneration(),
		nil,
		request.Actor.UserID,
		&generationOutputContext{
			engineVersionID: engineVersion.ID,
			step:            step,
		},
	)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(
		attribute.String("generation.id", generation.ID.String()),
		attribute.Bool("generation.created", response.GetCreated()),
	)

	return otel.ReportSuccess(span, &GenerationSubmitResult{
		Generation: generation,
		Created:    response.GetCreated(),
	}), nil
}
