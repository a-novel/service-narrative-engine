package core

import (
	"encoding/json"
	"errors"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

// validatePartialContent checks caller input against a content contract with presence constraints
// relaxed, then applies semantic constraints that JSON Schema cannot express.
func validatePartialContent(
	schema *lib.ContentSchema,
	value json.RawMessage,
	validateSemantics func(map[string]any) error,
) error {
	instance, err := schema.ValidatePartial(value)
	if err != nil {
		// The schema is service- or Engine-owned, so its failure is not a client error.
		if errors.Is(err, lib.ErrContentSchemaInvalid) {
			return errors.Join(ErrEngineDefinitionInvalid, err)
		}

		return errors.Join(ErrInvalidRequest, err)
	}

	if validateSemantics == nil {
		return nil
	}

	err = validateSemantics(instance)
	if err != nil {
		return errors.Join(ErrInvalidRequest, err)
	}

	return nil
}

// validateCompleteContent checks generated content against its full contract, then applies
// semantic constraints that JSON Schema cannot express.
func validateCompleteContent(
	schema *lib.ContentSchema,
	value json.RawMessage,
	validateSemantics func(map[string]any) error,
) error {
	instance, err := schema.ValidateComplete(value)
	if errors.Is(err, lib.ErrContentSchemaInvalid) {
		return errors.Join(ErrEngineDefinitionInvalid, err)
	}

	if err != nil {
		return err
	}

	if validateSemantics == nil {
		return nil
	}

	return validateSemantics(instance)
}
