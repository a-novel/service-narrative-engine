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

// generationTargetDefinition is the trusted prompt and content contract
// selected for one Narrative generation target.
type generationTargetDefinition struct {
	contentDefinition

	Target         GenerationTarget
	PromptTemplate string
}

// contentDefinition binds a JSON Schema to the semantic checks shared by saved
// content and generated proposals.
type contentDefinition struct {
	schema            *lib.ContentSchema
	validateSemantics func(map[string]any) error
}

// validatePartial classifies malformed definitions separately from invalid
// caller content before running semantic checks on the decoded document.
func (definition contentDefinition) validatePartial(value json.RawMessage) error {
	instance, err := definition.schema.ValidatePartial(value)
	if err != nil {
		if errors.Is(err, lib.ErrContentSchemaInvalid) {
			return errors.Join(ErrEngineDefinitionInvalid, err)
		}

		return errors.Join(ErrInvalidRequest, err)
	}

	if definition.validateSemantics == nil {
		return nil
	}

	err = definition.validateSemantics(instance)
	if err != nil {
		return errors.Join(ErrInvalidRequest, err)
	}

	return nil
}

// validateComplete preserves definition failures for service diagnostics and
// leaves proposal failures for the generation boundary to classify.
func (definition contentDefinition) validateComplete(value json.RawMessage) error {
	instance, err := definition.schema.ValidateComplete(value)
	if errors.Is(err, lib.ErrContentSchemaInvalid) {
		return errors.Join(ErrEngineDefinitionInvalid, err)
	}

	if err != nil {
		return err
	}

	if definition.validateSemantics == nil {
		return nil
	}

	return definition.validateSemantics(instance)
}

// loadGenerationTarget resolves the trusted prompt and content contract for one
// static target or Engine-defined step.
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
			Target:            target,
			PromptTemplate:    ideaGenerationPrompt,
			contentDefinition: ideaContentDefinition,
		}), nil
	case GenerationTargetKindManuscript:
		return otel.ReportSuccess(span, &generationTargetDefinition{
			Target:            target,
			PromptTemplate:    manuscriptGenerationPrompt,
			contentDefinition: manuscriptContentDefinition,
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
			contentDefinition: contentDefinition{
				schema: lib.NewContentSchema(
					step.OutputSchema,
					schemas.ContentDocumentMaxBytes,
				),
			},
		}), nil
	default:
		return nil, otel.ReportError(
			span,
			fmt.Errorf("%w: unknown generation target %q", ErrInvalidRequest, target.Kind),
		)
	}
}
