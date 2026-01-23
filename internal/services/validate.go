package services

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

var ErrInvalidRequest = errors.New("invalid request")

// ValidateNotBlank is a custom validator that ensures a string is not empty after trimming whitespace.
// Use the "notblank" tag to apply it.
func ValidateNotBlank(fl validator.FieldLevel) bool {
	return strings.TrimSpace(fl.Field().String()) != ""
}

func init() {
	err := validate.RegisterValidation("notblank", ValidateNotBlank)
	if err != nil {
		panic(err)
	}
}
