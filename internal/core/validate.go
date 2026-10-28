package core

import (
	"errors"
	"slices"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// ErrInvalidRequest is returned by a service Exec when its request fails struct validation.
var ErrInvalidRequest = errors.New("invalid request")

// ValidateNotBlank reports whether a string holds a non-whitespace character. The package registers
// it as the "notblank" struct tag.
func ValidateNotBlank(fl validator.FieldLevel) bool {
	return strings.TrimSpace(fl.Field().String()) != ""
}

// ValidateLanguage reports whether a string is one of [ProjectLanguages]. The package registers it as
// the "language" struct tag.
func ValidateLanguage(fl validator.FieldLevel) bool {
	return slices.Contains(ProjectLanguages, fl.Field().String())
}

// ValidateLayerKind reports whether a string names a layer kind that has a content schema. The package
// registers it as the "layerkind" struct tag.
func ValidateLayerKind(fl validator.FieldLevel) bool {
	_, err := schemas.Content(fl.Field().String())

	return err == nil
}

// ValidateUniqueCells reports whether a list of [CellsWriteItem] names each pair of a scene and a layer
// once. The package registers it as the "uniquecells" struct tag.
func ValidateUniqueCells(fl validator.FieldLevel) bool {
	cells, ok := fl.Field().Interface().([]CellsWriteItem)
	if !ok {
		return false
	}

	seen := make(map[[2]uuid.UUID]bool, len(cells))

	for _, cell := range cells {
		key := [2]uuid.UUID{cell.SceneID, cell.LayerID}
		if seen[key] {
			return false
		}

		seen[key] = true
	}

	return true
}

func init() {
	for tag, fn := range map[string]validator.Func{
		"notblank":    ValidateNotBlank,
		"language":    ValidateLanguage,
		"layerkind":   ValidateLayerKind,
		"uniquecells": ValidateUniqueCells,
	} {
		err := validate.RegisterValidation(tag, fn)
		if err != nil {
			panic(err)
		}
	}
}
