package core

import (
	"errors"
	"regexp"
	"slices"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"

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

// ValidateProjectLabel reports whether a genre of a project holds no whitespace at either end and no NUL, which
// the database driver drops on write. The package registers it as the "projectlabel" struct tag.
func ValidateProjectLabel(fl validator.FieldLevel) bool {
	label := fl.Field().String()

	return strings.TrimSpace(label) == label && !strings.ContainsRune(label, 0)
}

// projectTagPattern matches a tag of a project in lower kebab case.
var projectTagPattern = regexp.MustCompile(`^[\p{Ll}\p{Lm}\p{Lo}\p{M}\p{Nd}]+(-[\p{Ll}\p{Lm}\p{Lo}\p{M}\p{Nd}]+)*$`)

// ValidateProjectTag reports whether a tag of a project is in lower kebab case, as "années-50" is: runs of
// lowercase, modifier or caseless letters, marks or digits, from any script, joined by single hyphens. Marks keep
// whole the words of scripts that write with them, as Thai or Hindi do. The tag must be precomposed (NFC), so an
// accented letter has a single spelling. The package registers it as the "projecttag" struct tag.
func ValidateProjectTag(fl validator.FieldLevel) bool {
	tag := fl.Field().String()

	return projectTagPattern.MatchString(tag) && norm.NFC.IsNormalString(tag)
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
		"projectlabel":   ValidateProjectLabel,
		"projecttag":     ValidateProjectTag,
	}

	for tag, fn := range validations {
		err := validate.RegisterValidation(tag, fn)
		if err != nil {
			panic(err)
		}
	}
}
