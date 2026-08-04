package core

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// ErrInvalidRequest is joined into the error returned by a service's Exec when
// the request fails struct validation. Callers match against it to map the
// failure to a client error.
var ErrInvalidRequest = errors.New("invalid request")

// ValidateNotBlank reports whether a string holds a non-whitespace character.
// Apply it with the "notblank" struct tag.
func ValidateNotBlank(fl validator.FieldLevel) bool {
	return strings.TrimSpace(fl.Field().String()) != ""
}

// ValidateActor reports whether a request carries an authenticated user.
// Apply it with the "actor" struct tag.
func ValidateActor(fl validator.FieldLevel) bool {
	actor, ok := fl.Field().Interface().(Actor)

	return ok && actor.UserID != uuid.Nil
}

// ValidateGenerationTarget enforces the fields that identify dynamic step
// targets and excludes those fields from static Idea and Manuscript targets.
func ValidateGenerationTarget(sl validator.StructLevel) {
	target, ok := sl.Current().Interface().(GenerationTarget)
	if !ok {
		return
	}

	switch target.Kind {
	case GenerationTargetKindStep:
		if target.EngineVersionID == uuid.Nil {
			sl.ReportError(target.EngineVersionID, "EngineVersionID", "engineVersionID", "required", "")
		}

		if target.StepKey == "" {
			sl.ReportError(target.StepKey, "StepKey", "stepKey", "required", "")
		}
	case GenerationTargetKindIdea, GenerationTargetKindManuscript:
		if target.EngineVersionID != uuid.Nil {
			sl.ReportError(target.EngineVersionID, "EngineVersionID", "engineVersionID", "excluded", "")
		}

		if target.StepKey != "" {
			sl.ReportError(target.StepKey, "StepKey", "stepKey", "excluded", "")
		}
	}
}

func init() {
	err := validate.RegisterValidation("notblank", ValidateNotBlank)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("actor", ValidateActor)
	if err != nil {
		panic(err)
	}

	validate.RegisterStructValidation(ValidateGenerationTarget, GenerationTarget{})
}
