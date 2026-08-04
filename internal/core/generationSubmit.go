package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	errGenerationSavedStepMissing  = errors.New("latest step selection returned a nil value")
	errGenerationManuscriptMissing = errors.New("latest Manuscript selection returned a nil value")
)

// StepValueSelectLatestDao retrieves the current saved value for every logical step key.
type StepValueSelectLatestDao interface {
	Exec(
		ctx context.Context,
		request *dao.StepValueSelectLatestRequest,
	) ([]*dao.StepValue, error)
}

// ManuscriptSelectLatestDao retrieves the current saved Manuscript when one exists.
type ManuscriptSelectLatestDao interface {
	Exec(
		ctx context.Context,
		request *dao.ManuscriptSelectLatestRequest,
	) (*dao.Manuscript, error)
}

// GenerationSubmitRequest carries one partial target input and optional step-context replacements.
type GenerationSubmitRequest struct {
	Actor            Actor     `validate:"actor"`
	IdeaID           uuid.UUID `validate:"required"`
	Target           GenerationTarget
	Input            json.RawMessage             `validate:"required"`
	ContextOverrides []GenerationContextOverride `validate:"dive"`
	IdempotencyKey   string                      `validate:"required,notblank,max=256"`
}

// GenerationSubmit assembles and submits one self-contained project generation request.
type GenerationSubmit struct {
	projectAccess    ProjectAccessService
	engineVersionDao EngineVersionSelectDao
	stepValueDao     StepValueSelectLatestDao
	manuscriptDao    ManuscriptSelectLatestDao
	genai            servicegenai.Client
}

// NewGenerationSubmit creates the narrative generation submit service.
func NewGenerationSubmit(
	projectAccess ProjectAccessService,
	engineVersionDao EngineVersionSelectDao,
	stepValueDao StepValueSelectLatestDao,
	manuscriptDao ManuscriptSelectLatestDao,
	genai servicegenai.Client,
) *GenerationSubmit {
	return &GenerationSubmit{
		projectAccess:    projectAccess,
		engineVersionDao: engineVersionDao,
		stepValueDao:     stepValueDao,
		manuscriptDao:    manuscriptDao,
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
		attribute.String("generation.target_kind", string(request.Target.Kind)),
		attribute.String("generation.owner_id", request.Actor.UserID.String()),
	)

	if request.Target.Kind == GenerationTargetKindStep {
		span.SetAttributes(
			attribute.String("engine_version.id", request.Target.EngineVersionID.String()),
			attribute.String("engine.step_key", request.Target.StepKey),
		)
	}

	idea, err := service.projectAccess.Exec(ctx, &ProjectAccessRequest{
		Actor:  request.Actor,
		IdeaID: request.IdeaID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("access project: %w", err))
	}

	definition, err := loadGenerationTarget(ctx, service.engineVersionDao, request.Target)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	err = definition.validatePartial(request.Input)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("target input: %w", err))
	}

	overrides, excludedStepKeys, err := service.loadContextOverrides(
		ctx,
		request.ContextOverrides,
	)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	savedSteps, err := service.stepValueDao.Exec(ctx, &dao.StepValueSelectLatestRequest{
		IdeaID:          request.IdeaID,
		ExcludeStepKeys: excludedStepKeys,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select latest step values: %w", err))
	}

	contextSteps := make([]generationContextStep, 0, len(savedSteps)+len(overrides))
	for _, savedStep := range savedSteps {
		if savedStep == nil {
			return nil, otel.ReportError(span, fmt.Errorf("select latest step values: %w", errGenerationSavedStepMissing))
		}

		contextSteps = append(contextSteps, generationContextStep{
			EngineVersionID: savedStep.EngineVersionID,
			StepKey:         savedStep.StepKey,
			Value:           savedStep.Value,
		})
	}

	contextSteps = append(contextSteps, overrides...)
	sort.Slice(contextSteps, func(left int, right int) bool {
		return contextSteps[left].StepKey < contextSteps[right].StepKey
	})

	var manuscriptValue json.RawMessage

	manuscript, err := service.manuscriptDao.Exec(ctx, &dao.ManuscriptSelectLatestRequest{
		IdeaID: request.IdeaID,
	})
	if errors.Is(err, dao.ErrManuscriptSelectLatestNotFound) {
		err = nil
	} else if err == nil {
		if manuscript == nil {
			return nil, otel.ReportError(span, fmt.Errorf("select latest Manuscript: %w", errGenerationManuscriptMissing))
		}

		manuscriptValue = manuscript.Value
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select latest Manuscript: %w", err))
	}

	payload, err := buildGenerationPayload(
		definition,
		request.Input,
		idea,
		contextSteps,
		manuscriptValue,
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("build generation payload: %w", err))
	}

	idempotencyKey, err := deriveGenerationIdempotencyKey(
		request.IdempotencyKey,
		request.IdeaID,
		request.Target,
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
		&generationOutputContext{definition: definition},
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

// loadContextOverrides validates each client replacement against its selected
// Engine step and returns the step keys that must be excluded from the
// server-loaded context.
func (service *GenerationSubmit) loadContextOverrides(
	ctx context.Context,
	overrides []GenerationContextOverride,
) ([]generationContextStep, []string, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.GenerationSubmit.loadContextOverrides")
	defer span.End()

	contextSteps := make([]generationContextStep, 0, len(overrides))
	excludedStepKeys := make([]string, 0, len(overrides))

	duplicates := lo.FindDuplicates(lo.Map(overrides, func(
		override GenerationContextOverride,
		_ int,
	) string {
		return override.StepKey
	}))
	if len(duplicates) != 0 {
		return nil, nil, otel.ReportError(
			span,
			fmt.Errorf(
				"%w: duplicate context override for step %q",
				ErrInvalidRequest,
				duplicates[0],
			),
		)
	}

	for index := range overrides {
		override := &overrides[index]

		definition, err := loadGenerationTarget(ctx, service.engineVersionDao, GenerationTarget{
			Kind:            GenerationTargetKindStep,
			EngineVersionID: override.EngineVersionID,
			StepKey:         override.StepKey,
		})
		if err != nil {
			return nil, nil, otel.ReportError(
				span,
				fmt.Errorf("context override %d: %w", index, err),
			)
		}

		err = definition.validatePartial(override.Value)
		if err != nil {
			return nil, nil, otel.ReportError(
				span,
				fmt.Errorf("context override %d: %w", index, err),
			)
		}

		contextSteps = append(contextSteps, generationContextStep{
			EngineVersionID: override.EngineVersionID,
			StepKey:         override.StepKey,
			Value:           override.Value,
		})
		excludedStepKeys = append(excludedStepKeys, override.StepKey)
	}

	otel.ReportSuccessNoContent(span)

	return contextSteps, excludedStepKeys, nil
}
