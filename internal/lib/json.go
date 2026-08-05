package lib

import (
	"encoding/json"
	"errors"
)

var (
	// ErrJSONEmpty reports a missing JSON value.
	ErrJSONEmpty = errors.New("JSON value is empty")
	// ErrJSONInvalid reports malformed JSON.
	ErrJSONInvalid = errors.New("JSON value is invalid")
	// ErrJSONTooLarge reports a JSON value over its byte limit.
	ErrJSONTooLarge = errors.New("JSON value exceeds the size limit")
)

// ValidateJSON accepts any single JSON value within maxBytes.
// A non-positive maxBytes disables the byte limit.
func ValidateJSON(source []byte, maxBytes int) error {
	if len(source) == 0 {
		return ErrJSONEmpty
	}

	if maxBytes > 0 && len(source) > maxBytes {
		return ErrJSONTooLarge
	}

	if !json.Valid(source) {
		return ErrJSONInvalid
	}

	return nil
}
