package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

	var match *engineStepDefinition

	for index := range engine.Steps {
		step := &engine.Steps[index]
		if step.Key != key {
			continue
		}

		if match != nil {
			return nil, fmt.Errorf("%w: duplicate step key %q", ErrEngineDefinitionInvalid, key)
		}

		match = step
	}

	if match == nil {
		return nil, fmt.Errorf("%w: %q", ErrEngineStepNotFound, key)
	}

	if strings.TrimSpace(match.Key) == "" || strings.TrimSpace(match.PromptTemplate) == "" {
		return nil, fmt.Errorf("%w: step %q is incomplete", ErrEngineDefinitionInvalid, key)
	}

	schemaDefinition, err := loadContentSchema(match.OutputSchema)
	if err != nil {
		return nil, fmt.Errorf("%w: load output schema for step %q: %w", ErrEngineDefinitionInvalid, key, err)
	}

	match.contentSchemaDefinition = *schemaDefinition

	return match, nil
}

func (step *engineStepDefinition) validatePartialValue(value json.RawMessage) error {
	return step.validatePartial(value)
}
