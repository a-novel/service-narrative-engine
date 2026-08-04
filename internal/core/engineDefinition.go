package core

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/samber/lo"
)

var (
	// ErrEngineDefinitionInvalid reports an Engine Version definition that cannot
	// safely drive generation or local validation.
	ErrEngineDefinitionInvalid = errors.New("invalid engine definition")
	// ErrEngineStepNotFound reports a requested step absent from the Engine Version.
	ErrEngineStepNotFound = errors.New("engine step not found")
)

type engineDefinition struct {
	Steps []engineStepDefinition `json:"steps"`
}

type engineStepDefinition struct {
	contentSchemaDefinition

	Key            string          `json:"key"`
	PromptTemplate string          `json:"promptTemplate"`
	OutputSchema   json.RawMessage `json:"outputSchema"`
}

func selectEngineStep(definition json.RawMessage, key string) (*engineStepDefinition, error) {
	var engine engineDefinition

	err := json.Unmarshal(definition, &engine)
	if err != nil {
		return nil, fmt.Errorf("%w: decode definition: %w", ErrEngineDefinitionInvalid, err)
	}

	match, found := lo.Find(engine.Steps, func(step engineStepDefinition) bool {
		return step.Key == key
	})
	if !found {
		return nil, fmt.Errorf("%w: %q", ErrEngineStepNotFound, key)
	}

	return &match, nil
}

func (step *engineStepDefinition) loadOutputSchema() error {
	schemaDefinition, err := loadContentSchema(step.OutputSchema)
	if err != nil {
		return fmt.Errorf(
			"%w: load output schema for step %q: %w",
			ErrEngineDefinitionInvalid,
			step.Key,
			err,
		)
	}

	step.contentSchemaDefinition = *schemaDefinition

	return nil
}

func (step *engineStepDefinition) validatePartialValue(value json.RawMessage) error {
	return step.validatePartial(value)
}
