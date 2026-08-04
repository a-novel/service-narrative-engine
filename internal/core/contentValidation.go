package core

import (
	"encoding/json"
	"errors"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func validatePartialContent(
	schema *lib.ContentSchema,
	value json.RawMessage,
	validateSemantics func(map[string]any) error,
) error {
	instance, err := schema.ValidatePartial(value)

	err = validateContent(instance, err, validateSemantics)
	if err != nil && !errors.Is(err, ErrEngineDefinitionInvalid) {
		return errors.Join(ErrInvalidRequest, err)
	}

	return err
}

func validateCompleteContent(
	schema *lib.ContentSchema,
	value json.RawMessage,
	validateSemantics func(map[string]any) error,
) error {
	instance, err := schema.ValidateComplete(value)

	return validateContent(instance, err, validateSemantics)
}

func validateContent(
	instance map[string]any,
	err error,
	validateSemantics func(map[string]any) error,
) error {
	if errors.Is(err, lib.ErrContentSchemaInvalid) {
		return errors.Join(ErrEngineDefinitionInvalid, err)
	}

	if err != nil || validateSemantics == nil {
		return err
	}

	return validateSemantics(instance)
}
