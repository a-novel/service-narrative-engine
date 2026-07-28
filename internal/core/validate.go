package core

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
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

// ValidateMaxBytes reports whether a string's UTF-8 representation fits within
// the byte count carried by the "maxbytes" struct tag.
func ValidateMaxBytes(fl validator.FieldLevel) bool {
	maxBytes, err := strconv.Atoi(fl.Param())
	if err != nil {
		return false
	}

	return len(fl.Field().String()) <= maxBytes
}

func init() {
	err := validate.RegisterValidation("notblank", ValidateNotBlank)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("maxbytes", ValidateMaxBytes)
	if err != nil {
		panic(err)
	}
}
