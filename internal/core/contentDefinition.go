package core

import (
	"encoding/json"
	"errors"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

var errStaticContentDefinitionInvalid = errors.New("static content definition is invalid")

// contentDefinition binds a static JSON Schema to optional semantic checks.
type contentDefinition struct {
	schema            *lib.ContentSchema
	validateSemantics func(map[string]any) error
}

// validatePartial keeps service-owned schema failures distinct from invalid caller content.
func (definition contentDefinition) validatePartial(value json.RawMessage) error {
	instance, err := definition.schema.ValidatePartial(value)
	if errors.Is(err, lib.ErrContentSchemaInvalid) {
		return errors.Join(errStaticContentDefinitionInvalid, err)
	}

	if err != nil {
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
