package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
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
	Key            string          `json:"key"`
	PromptTemplate string          `json:"promptTemplate"`
	OutputSchema   json.RawMessage `json:"outputSchema"`
	resolvedSchema *jsonschema.Resolved
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

	var schema jsonschema.Schema

	err = json.Unmarshal(match.OutputSchema, &schema)
	if err != nil {
		return nil, fmt.Errorf("%w: decode output schema for step %q: %w", ErrEngineDefinitionInvalid, key, err)
	}

	match.resolvedSchema, err = schema.Resolve(nil)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve output schema for step %q: %w", ErrEngineDefinitionInvalid, key, err)
	}

	return match, nil
}

func (step *engineStepDefinition) validateValue(value json.RawMessage) error {
	var instance any

	err := json.Unmarshal(value, &instance)
	if err != nil {
		return fmt.Errorf("decode JSON value: %w", err)
	}

	err = step.resolvedSchema.Validate(instance)
	if err != nil {
		return fmt.Errorf("validate JSON value: %w", err)
	}

	return nil
}
