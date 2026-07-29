package core

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

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

	err := validateGenerationTarget(target)
	if err != nil {
		return nil, err
	}

	switch target.Kind {
	case GenerationTargetKindIdea:
		return loadStaticGenerationTarget(target, ideaGenerationPrompt, ideaOutputSchema)
	case GenerationTargetKindManuscript:
		return loadStaticGenerationTarget(target, manuscriptGenerationPrompt, manuscriptOutputSchema)
	case GenerationTargetKindStep:
		engineVersion, selectErr := engineVersionDao.Exec(ctx, &dao.EngineVersionSelectRequest{
			ID: target.EngineVersionID,
		})
		if errors.Is(selectErr, dao.ErrEngineVersionSelectNotFound) {
			selectErr = errors.Join(selectErr, ErrEngineVersionNotFound)
		}

		if selectErr != nil {
			return nil, fmt.Errorf("select Engine Version: %w", selectErr)
		}

		step, selectErr := selectEngineStep(engineVersion.Definition, target.StepKey)
		if selectErr != nil {
			return nil, selectErr
		}

		return &generationTargetDefinition{
			Target:                  target,
			PromptTemplate:          step.PromptTemplate,
			contentSchemaDefinition: step.contentSchemaDefinition,
		}, nil
	default:
		return nil, fmt.Errorf("%w: unknown generation target %q", ErrInvalidRequest, target.Kind)
	}
}

func loadStaticGenerationTarget(
	target GenerationTarget,
	prompt string,
	outputSchema []byte,
) (*generationTargetDefinition, error) {
	schema, err := loadContentSchema(outputSchema)
	if err != nil {
		return nil, fmt.Errorf("load %s schema: %w", target.Kind, err)
	}

	return &generationTargetDefinition{
		Target:                  target,
		PromptTemplate:          prompt,
		contentSchemaDefinition: *schema,
	}, nil
}

func validateGenerationTarget(target GenerationTarget) error {
	switch target.Kind {
	case GenerationTargetKindIdea, GenerationTargetKindManuscript:
		if target.EngineVersionID != uuid.Nil || target.StepKey != "" {
			return fmt.Errorf(
				"%w: %s target cannot identify an Engine step",
				ErrInvalidRequest,
				target.Kind,
			)
		}
	case GenerationTargetKindStep:
		if target.EngineVersionID == uuid.Nil ||
			strings.TrimSpace(target.StepKey) == "" ||
			len([]rune(target.StepKey)) > 256 {
			return fmt.Errorf("%w: incomplete step target", ErrInvalidRequest)
		}
	default:
		return fmt.Errorf("%w: unknown generation target %q", ErrInvalidRequest, target.Kind)
	}

	return nil
}
