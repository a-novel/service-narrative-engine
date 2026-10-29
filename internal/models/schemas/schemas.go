package schemas

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
)

// files holds every schema of the package, read from the files beside this one at build time.
//
//go:embed *.schema.json
var files embed.FS

// baseURI anchors the relative references between the embedded files.
const baseURI = "file:///schemas/"

var (
	// ErrUnknownKind is returned by [Content] for a layer kind that has no content schema.
	ErrUnknownKind = errors.New("unknown layer kind")
	// ErrUnknownGenerationKind is returned by [Output] for a generation kind that has no output schema.
	ErrUnknownGenerationKind = errors.New("unknown generation kind")
	// ErrInvalidDocument is returned by [Schema.Validate] for a document that is not JSON, or that
	// breaks the schema.
	ErrInvalidDocument = errors.New("invalid document")
)

// Schema is one embedded JSON Schema. It compiles on its first use.
type Schema struct {
	resolve func() (*jsonschema.Resolved, error)
	strict  func() (json.RawMessage, error)
}

// newSchema returns the schema in the embedded file name, compiled on its first use.
func newSchema(name string) *Schema {
	return &Schema{
		resolve: sync.OnceValues(func() (*jsonschema.Resolved, error) { return resolve(name) }),
		strict:  sync.OnceValues(func() (json.RawMessage, error) { return bundleStrict(name) }),
	}
}

var (
	// contentSchemas maps each layer kind to the schema of its cells. The kind is the file name between
	// content. and .schema.json.
	contentSchemas = loadSchemas("content.")
	// outputSchemas maps each generation kind to the schema of the model's output. The kind is the file
	// name between output. and .schema.json.
	outputSchemas = loadSchemas("output.")
)

// Manuscript is the schema of an exported manuscript, format v1. Its scenes hold the blocks of the
// manuscript content schema.
var Manuscript = newSchema("manuscript.schema.json")

// Content returns the content schema of a layer kind.
func Content(kind string) (*Schema, error) {
	schema, ok := contentSchemas[kind]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
	}

	return schema, nil
}

// Output returns the schema a model's output follows for a generation kind. It holds every limit, and
// [Schema.Strict] gives the form the model is asked to follow.
func Output(kind string) (*Schema, error) {
	schema, ok := outputSchemas[kind]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownGenerationKind, kind)
	}

	return schema, nil
}

// Validate checks that document, a JSON value, follows the schema.
func (schema *Schema) Validate(document json.RawMessage) error {
	resolved, err := schema.resolve()
	if err != nil {
		return fmt.Errorf("resolve schema: %w", err)
	}

	var value any

	err = json.Unmarshal(document, &value)
	if err != nil {
		return errors.Join(err, ErrInvalidDocument)
	}

	err = resolved.Validate(value)
	if err != nil {
		return errors.Join(err, ErrInvalidDocument)
	}

	return nil
}

// Strict returns the schema as one JSON document in the subset of JSON Schema that OpenAI's strict
// structured outputs accept. The files it references become definitions of the document, and the limits
// that subset cannot express are left to [Schema.Validate].
func (schema *Schema) Strict() (json.RawMessage, error) {
	return schema.strict()
}

// loadSchemas finds every embedded schema whose file name starts with prefix, keyed by the rest of the
// name before .schema.json.
func loadSchemas(prefix string) map[string]*Schema {
	names, err := fs.Glob(files, prefix+"*.schema.json")
	if err != nil {
		panic(err)
	}

	loaded := make(map[string]*Schema, len(names))
	for _, name := range names {
		kind := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".schema.json")
		loaded[kind] = newSchema(name)
	}

	return loaded
}

// resolve compiles the schema in the embedded file name. A reference to another schema file loads it
// from the embedded files too, by its name relative to [baseURI].
func resolve(name string) (*jsonschema.Resolved, error) {
	schema, err := load(name)
	if err != nil {
		return nil, err
	}

	return schema.Resolve(&jsonschema.ResolveOptions{
		BaseURI: baseURI + name,
		Loader: func(uri *url.URL) (*jsonschema.Schema, error) {
			return load(path.Base(uri.Path))
		},
	})
}

// load decodes the embedded file name as a JSON Schema, without resolving its references.
func load(name string) (*jsonschema.Schema, error) {
	data, err := files.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read schema %s: %w", name, err)
	}

	schema := &jsonschema.Schema{}

	err = json.Unmarshal(data, schema)
	if err != nil {
		return nil, fmt.Errorf("decode schema %s: %w", name, err)
	}

	return schema, nil
}
