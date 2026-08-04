package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

const (
	ideaGenerationPrompt = "Complete the Idea from the partial input. Reconcile it with project " +
		"context while preserving explicit input."
	manuscriptGenerationPrompt = "Complete the Manuscript from the partial input. Reconcile it " +
		"with all project context while preserving explicit input."
)

type generationTargetDefinition struct {
	contentSchemaDefinition

	Target         GenerationTarget
	PromptTemplate string
}

func loadGenerationTarget(
	ctx context.Context,
	engineVersionDao EngineVersionSelectDao,
	target GenerationTarget,
) (*generationTargetDefinition, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.loadGenerationTarget")
	defer span.End()

	err := validate.Struct(target)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	switch target.Kind {
	case GenerationTargetKindIdea:
		definition, loadErr := loadStaticGenerationTarget(
			target,
			ideaGenerationPrompt,
			ideaOutputSchema,
		)
		if loadErr != nil {
			return nil, otel.ReportError(span, loadErr)
		}

		return otel.ReportSuccess(span, definition), nil
	case GenerationTargetKindManuscript:
		definition, loadErr := loadStaticGenerationTarget(
			target,
			manuscriptGenerationPrompt,
			manuscriptOutputSchema,
		)
		if loadErr != nil {
			return nil, otel.ReportError(span, loadErr)
		}

		return otel.ReportSuccess(span, definition), nil
	case GenerationTargetKindStep:
		engineVersion, selectErr := engineVersionDao.Exec(ctx, &dao.EngineVersionSelectRequest{
			ID: target.EngineVersionID,
		})
		if errors.Is(selectErr, dao.ErrEngineVersionSelectNotFound) {
			selectErr = errors.Join(selectErr, ErrEngineVersionNotFound)
		}

		if selectErr != nil {
			return nil, otel.ReportError(span, fmt.Errorf("select Engine Version: %w", selectErr))
		}

		step, selectErr := selectEngineStep(engineVersion.Definition, target.StepKey)
		if selectErr != nil {
			return nil, otel.ReportError(span, selectErr)
		}

		selectErr = step.loadOutputSchema()
		if selectErr != nil {
			return nil, otel.ReportError(span, selectErr)
		}

		return otel.ReportSuccess(span, &generationTargetDefinition{
			Target:                  target,
			PromptTemplate:          step.PromptTemplate,
			contentSchemaDefinition: step.contentSchemaDefinition,
		}), nil
	default:
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: unknown generation target %q", ErrInvalidRequest, target.Kind),
		)
	}
}

func loadStaticGenerationTarget(
	target GenerationTarget,
	prompt string,
	outputSchema []byte,
) (*generationTargetDefinition, error) {
	var (
		schema *contentSchemaDefinition
		err    error
	)

	if target.Kind == GenerationTargetKindManuscript {
		schema, err = loadManuscriptContentSchema()
	} else {
		schema, err = loadContentSchema(outputSchema)
	}

	if err != nil {
		return nil, fmt.Errorf("load %s schema: %w", target.Kind, err)
	}

	return &generationTargetDefinition{
		Target:                  target,
		PromptTemplate:          prompt,
		contentSchemaDefinition: *schema,
	}, nil
}
