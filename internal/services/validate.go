package services

import (
	"errors"

	"github.com/go-playground/validator/v10"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/lib"
	"github.com/a-novel/service-narrative-engine/internal/models"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrInvalidData    = errors.New("invalid data")
)

func ValidateLang(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	for _, lang := range config.KnownLangs {
		if val == lang {
			return true
		}
	}

	return false
}

func ValidateModule(fl validator.FieldLevel) bool {
	val := fl.Field().String()

	return lib.ModuleStringRegexp.MatchString(val)
}

func ValidateModuleName(fl validator.FieldLevel) bool {
	val := fl.Field().String()

	return lib.ModuleNameRegexp.MatchString(val)
}

func ValidateModuleVersion(fl validator.FieldLevel) bool {
	val := fl.Field().String()

	return lib.ModuleVersionRegexp.MatchString(val)
}

func ValidateSource(fl validator.FieldLevel) bool {
	val := fl.Field().String()

	for _, src := range models.KnownSchemaSources {
		if val == string(src) {
			return true
		}
	}

	return false
}

func init() {
	err := validate.RegisterValidation("langs", ValidateLang)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("module", ValidateModule)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("schemaSource", ValidateSource)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("moduleName", ValidateModuleName)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("moduleVersion", ValidateModuleVersion)
	if err != nil {
		panic(err)
	}
}
