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

// ValidateGenerationKind reports whether a string is one of [GenerationKinds]. The package registers it as
// the "generationkind" struct tag.
func ValidateGenerationKind(fl validator.FieldLevel) bool {
	return slices.Contains(GenerationKinds, fl.Field().String())
}

// ValidateTargetMode reports whether a string is one of [TargetModes]. The package registers it as the
// "targetmode" struct tag.
func ValidateTargetMode(fl validator.FieldLevel) bool {
	return slices.Contains(TargetModes, fl.Field().String())
}

// ValidateSceneField reports whether a string is one of [SceneFields]. The package registers it as the
// "scenefield" struct tag.
func ValidateSceneField(fl validator.FieldLevel) bool {
	return slices.Contains(SceneFields, fl.Field().String())
}

// ValidateStreamSection reports whether a string is one of [ProjectStreamSections]. The package registers it as
// the "streamsection" struct tag.
func ValidateStreamSection(fl validator.FieldLevel) bool {
	return slices.Contains(ProjectStreamSections, fl.Field().String())
}

// ValidateProjectPicture reports whether a string names a picture a project may hold: [ProjectPictureLogo] or
// [ProjectPictureBackground]. The package registers it as the "projectpicture" struct tag.
func ValidateProjectPicture(fl validator.FieldLevel) bool {
	_, ok := projectPictureContracts[fl.Field().String()]

	return ok
}

// ValidatePictureBytes reports whether an encoded picture fits the byte cap of the picture its struct's
// Picture field names. The package registers it as the "picturebytes" struct tag.
func ValidatePictureBytes(fl validator.FieldLevel) bool {
	contract, ok := projectPictureContracts[fl.Parent().FieldByName("Picture").String()]

	return ok && fl.Field().Len() <= contract.maxBytes
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
	validations := map[string]validator.Func{
		"notblank":       ValidateNotBlank,
		"language":       ValidateLanguage,
		"layerkind":      ValidateLayerKind,
		"uniquecells":    ValidateUniqueCells,
		"generationkind": ValidateGenerationKind,
		"targetmode":     ValidateTargetMode,
		"scenefield":     ValidateSceneField,
		"streamsection":  ValidateStreamSection,
		"projectpicture": ValidateProjectPicture,
		"picturebytes":   ValidatePictureBytes,
	}

	for tag, fn := range validations {
		err := validate.RegisterValidation(tag, fn)
		if err != nil {
			panic(err)
		}
	}
}
