package schemas

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	errStrictReference  = errors.New("unsupported reference")
	errStrictDefinition = errors.New("duplicate definition")
	errStrictSchema     = errors.New("schema is not an object")
)

// strictBundle rewrites an embedded schema and the files it references into one document in the subset of
// JSON Schema that OpenAI's strict structured outputs accept.
//
// The root file keeps the names of its definitions. A referenced file becomes the definition named after
// it, block for block.schema.json, and its own definitions take that name as a prefix, as in
// block-narration.
type strictBundle struct {
	root  string
	defs  map[string]any
	files map[string]bool
}

// bundleStrict returns the embedded file name as one document in strict mode's subset. See [Schema.Strict].
func bundleStrict(name string) (json.RawMessage, error) {
	bundle := &strictBundle{root: name, defs: map[string]any{}, files: map[string]bool{name: true}}

	root, err := bundle.file(name)
	if err != nil {
		return nil, fmt.Errorf("bundle %s: %w", name, err)
	}

	if len(bundle.defs) > 0 {
		root["$defs"] = bundle.defs
	}

	return json.Marshal(root)
}

// file rewrites the embedded file name and returns its root schema. Its definitions join the bundle.
func (bundle *strictBundle) file(name string) (map[string]any, error) {
	data, err := files.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read schema %s: %w", name, err)
	}

	var document map[string]any

	err = json.Unmarshal(data, &document)
	if err != nil {
		return nil, fmt.Errorf("decode schema %s: %w", name, err)
	}

	defs, err := asSchema(document["$defs"], map[string]any{})
	if err != nil {
		return nil, err
	}

	delete(document, "$defs")

	for def, schema := range defs {
		lowered, err := bundle.lowerOne(schema, name)
		if err != nil {
			return nil, err
		}

		err = bundle.define(bundle.defName(name, def), lowered)
		if err != nil {
			return nil, err
		}
	}

	return bundle.lower(document, name)
}

// define adds a rewritten schema to the definitions of the bundle.
func (bundle *strictBundle) define(def string, schema map[string]any) error {
	if _, ok := bundle.defs[def]; ok {
		return fmt.Errorf("%w: %s", errStrictDefinition, def)
	}

	bundle.defs[def] = schema

	return nil
}

// lower rewrites schema, a part of the file name, into strict mode's subset. It drops the annotations and
// the limits strict mode lacks, writes a const as a one-value enum and a oneOf as an anyOf, and points
// each reference at a definition of the bundle.
func (bundle *strictBundle) lower(schema map[string]any, name string) (map[string]any, error) {
	lowered := make(map[string]any, len(schema))

	for keyword, value := range schema {
		var err error

		switch keyword {
		case "$schema", "$id", "title", "maxLength", "minLength":
		case "const":
			lowered["enum"] = []any{value}
		case "$ref":
			lowered[keyword], err = bundle.ref(value, name)
		case "oneOf", "anyOf":
			lowered["anyOf"], err = bundle.lowerList(value, name)
		case "properties":
			lowered[keyword], err = bundle.lowerMap(value, name)
		case "items":
			lowered[keyword], err = bundle.lowerOne(value, name)
		default:
			lowered[keyword] = value
		}

		if err != nil {
			return nil, err
		}
	}

	return lowered, nil
}

// lowerOne lowers value, one schema.
func (bundle *strictBundle) lowerOne(value any, name string) (map[string]any, error) {
	schema, err := asSchema(value, nil)
	if err != nil {
		return nil, err
	}

	return bundle.lower(schema, name)
}

// lowerList lowers value, a list of schemas.
func (bundle *strictBundle) lowerList(value any, name string) ([]any, error) {
	list, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: %v", errStrictSchema, value)
	}

	lowered := make([]any, len(list))

	for index, item := range list {
		schema, err := bundle.lowerOne(item, name)
		if err != nil {
			return nil, err
		}

		lowered[index] = schema
	}

	return lowered, nil
}

// lowerMap lowers value, a map of schemas such as the properties of an object.
func (bundle *strictBundle) lowerMap(value any, name string) (map[string]any, error) {
	schemas, err := asSchema(value, nil)
	if err != nil {
		return nil, err
	}

	lowered := make(map[string]any, len(schemas))

	for key, item := range schemas {
		lowered[key], err = bundle.lowerOne(item, name)
		if err != nil {
			return nil, err
		}
	}

	return lowered, nil
}

// ref points value, a reference made in the file name, at a definition of the bundle. The first reference
// to another file adds that file to the bundle.
func (bundle *strictBundle) ref(value any, name string) (string, error) {
	reference, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%w: %v", errStrictReference, value)
	}

	file, pointer, _ := strings.Cut(reference, "#")
	if file == "" {
		file = name
	}

	def, isDef := strings.CutPrefix(pointer, "/$defs/")
	if pointer != "" && !isDef {
		return "", fmt.Errorf("%w: %s", errStrictReference, reference)
	}

	if !bundle.files[file] {
		bundle.files[file] = true

		root, err := bundle.file(file)
		if err != nil {
			return "", err
		}

		err = bundle.define(bundle.defName(file, ""), root)
		if err != nil {
			return "", err
		}
	}

	if file == bundle.root && !isDef {
		return "#", nil
	}

	return "#/$defs/" + bundle.defName(file, def), nil
}

// defName names a definition of the bundle: def of the file name, or the file's root when def is empty.
func (bundle *strictBundle) defName(file, def string) string {
	if file == bundle.root {
		return def
	}

	stem := strings.ReplaceAll(strings.TrimSuffix(file, ".schema.json"), ".", "-")
	if def == "" {
		return stem
	}

	return stem + "-" + def
}

// asSchema reads value as a schema object, or returns fallback when value is absent and fallback is set.
func asSchema(value any, fallback map[string]any) (map[string]any, error) {
	if value == nil && fallback != nil {
		return fallback, nil
	}

	schema, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: %v", errStrictSchema, value)
	}

	return schema, nil
}
