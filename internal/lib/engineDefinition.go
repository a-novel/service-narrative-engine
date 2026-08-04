package lib

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	// ErrEngineDefinitionInvalid reports an Engine definition that cannot be decoded.
	ErrEngineDefinitionInvalid = errors.New("invalid engine definition")
	// ErrEngineStepNotFound reports a requested step absent from an Engine definition.
	ErrEngineStepNotFound = errors.New("engine step not found")
)

type engineDefinition struct {
	Steps []EngineStepDefinition `json:"steps"`
}

// EngineStepDefinition is the generation contract selected from an Engine definition.
type EngineStepDefinition struct {
	Key            string          `json:"key"`
	PromptTemplate string          `json:"promptTemplate"`
	OutputSchema   json.RawMessage `json:"outputSchema"`
}

// SelectEngineStep decodes an Engine definition and returns its named step.
func SelectEngineStep(definition json.RawMessage, key string) (*EngineStepDefinition, error) {
	var engine engineDefinition

	err := json.Unmarshal(definition, &engine)
	if err != nil {
		return nil, fmt.Errorf("%w: decode definition: %w", ErrEngineDefinitionInvalid, err)
	}

	for index := range engine.Steps {
		if engine.Steps[index].Key == key {
			return &engine.Steps[index], nil
		}
	}

	return nil, fmt.Errorf("%w: %q", ErrEngineStepNotFound, key)
}
