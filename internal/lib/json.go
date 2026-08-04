package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ErrJSONMultipleValues reports input that contains more than one JSON value.
var ErrJSONMultipleValues = errors.New("multiple JSON values")

// DecodeJSONStrict decodes exactly one JSON value and rejects unknown object
// fields when destination is a struct.
func DecodeJSONStrict(source []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.DisallowUnknownFields()

	err := decoder.Decode(destination)
	if err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}

	var trailing any

	err = decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("decode trailing JSON: %w", err)
	}

	return ErrJSONMultipleValues
}
