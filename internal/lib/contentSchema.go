package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"unicode/utf8"

	"github.com/google/jsonschema-go/jsonschema"
)

var (
	// ErrContentSchemaInvalid reports a schema that cannot be prepared for validation.
	ErrContentSchemaInvalid = errors.New("invalid content schema")

	errContentDocumentEmpty         = errors.New("json document is empty")
	errContentDocumentInvalid       = errors.New("json document is invalid")
	errContentDocumentNotObject     = errors.New("json value must be an object")
	errContentDocumentSchemaInvalid = errors.New("json document does not match its schema")
	errContentDocumentTooLarge      = errors.New("json document exceeds the size limit")
)

// ContentSchema validates bounded object documents against one JSON Schema.
// The schema is compiled once on its first use.
type ContentSchema struct {
	maxBytes int
	resolve  func() (*jsonschema.Resolved, error)
}

// NewContentSchema creates a lazy validator. A non-positive maxBytes disables
// the document-size limit.
func NewContentSchema(schemaJSON json.RawMessage, maxBytes int) *ContentSchema {
	schemaJSON = bytes.Clone(schemaJSON)

	return &ContentSchema{
		maxBytes: maxBytes,
		resolve: sync.OnceValues(func() (*jsonschema.Resolved, error) {
			return resolveContentSchema(schemaJSON)
		}),
	}
}

// resolveContentSchema validates references and compiles one immutable schema
// for repeated document validation.
func resolveContentSchema(schemaJSON json.RawMessage) (*jsonschema.Resolved, error) {
	var schema jsonschema.Schema

	err := json.Unmarshal(schemaJSON, &schema)
	if err != nil {
		return nil, fmt.Errorf("decode JSON Schema: %w", err)
	}

	resolved, err := schema.Resolve(nil)
	if err != nil {
		return nil, fmt.Errorf("resolve JSON Schema: %w", err)
	}

	return resolved, nil
}

// Validate applies the schema to one bounded object document and returns its
// decoded tree.
func (schema *ContentSchema) Validate(value json.RawMessage) (map[string]any, error) {
	resolved, err := schema.resolve()
	if err != nil {
		return nil, fmt.Errorf("%w: prepare schema: %w", ErrContentSchemaInvalid, err)
	}

	instance, err := decodeContentDocument(value, schema.maxBytes)
	if err != nil {
		return nil, err
	}

	err = resolved.Validate(instance)
	if err != nil {
		// jsonschema-go includes rejected values in errors; expose an opaque error.
		return nil, errContentDocumentSchemaInvalid
	}

	return instance, nil
}

// decodeContentDocument enforces byte and UTF-8 bounds and accepts exactly one
// JSON object as content.
func decodeContentDocument(value json.RawMessage, maxBytes int) (map[string]any, error) {
	if maxBytes > 0 && len(value) > maxBytes {
		return nil, fmt.Errorf(
			"%w: contains %d bytes, limit is %d",
			errContentDocumentTooLarge,
			len(value),
			maxBytes,
		)
	}

	if !utf8.Valid(value) {
		return nil, errContentDocumentInvalid
	}

	value = bytes.TrimSpace(value)
	if len(value) == 0 {
		return nil, errContentDocumentEmpty
	}

	var instance map[string]any

	err := json.Unmarshal(value, &instance)
	if err != nil {
		return nil, errContentDocumentInvalid
	}

	if instance == nil {
		return nil, errContentDocumentNotObject
	}

	return instance, nil
}
