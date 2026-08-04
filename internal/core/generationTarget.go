package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/lib"
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

const (
	ideaGenerationPrompt = "Complete the Idea from the partial input. Reconcile it with project " +
		"context while preserving explicit input."
	manuscriptGenerationPrompt = "Complete the Manuscript from the partial input. Reconcile it " +
		"with all project context while preserving explicit input."
)

type generationTargetDefinition struct {
	Target            GenerationTarget
	PromptTemplate    string
	schema            *lib.ContentSchema
	validateSemantics func(map[string]any) error
}

func (definition *generationTargetDefinition) validatePartial(value json.RawMessage) error {
	return validatePartialContent(definition.schema, value, definition.validateSemantics)
}

func (definition *generationTargetDefinition) validateComplete(value json.RawMessage) error {
	return validateCompleteContent(definition.schema, value, definition.validateSemantics)
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
		return otel.ReportSuccess(span, &generationTargetDefinition{
			Target:         target,
			PromptTemplate: ideaGenerationPrompt,
			schema:         ideaContentSchema,
		}), nil
	case GenerationTargetKindManuscript:
		return otel.ReportSuccess(span, &generationTargetDefinition{
			Target:            target,
			PromptTemplate:    manuscriptGenerationPrompt,
			schema:            manuscriptContentSchema,
			validateSemantics: validateManuscriptContent,
		}), nil
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

		step, selectErr := lib.SelectEngineStep(engineVersion.Definition, target.StepKey)
		if selectErr != nil {
			return nil, otel.ReportError(span, selectErr)
		}

		return otel.ReportSuccess(span, &generationTargetDefinition{
			Target:         target,
			PromptTemplate: step.PromptTemplate,
			schema: lib.NewContentSchema(
				step.OutputSchema,
				schemas.ContentDocumentMaxBytes,
			),
		}), nil
	default:
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: unknown generation target %q", ErrInvalidRequest, target.Kind),
		)
	}
}
