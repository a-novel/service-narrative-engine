package lib

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/samber/lo"
)

var (
	ErrSchemaNotGeneratable = errors.New("schema is not generatable")
	ErrInvalidSchema        = errors.New("invalid schema")
	ErrInvalidSchemaFormat  = errors.New("invalid schema format")
)

type SchemaBuildFlag int

const (
	// SchemaBuildFlagAI set during AI content generation.
	SchemaBuildFlagAI SchemaBuildFlag = iota
	// SchemaBuildFlagPartial set during schema edition.
	SchemaBuildFlagPartial
)

type SchemaValidateFlag int

const (
	// SchemaValidateFlagGenerate makes sure the schema definition is compatible with AI generation.
	SchemaValidateFlagGenerate SchemaValidateFlag = iota
)

type SchemaType []string

func NewSchemaType(typ ...string) SchemaType {
	return append(SchemaType{}, typ...)
}

func (raw SchemaType) MarshalJSON() ([]byte, error) {
	switch len(raw) {
	case 0:
		return json.Marshal(nil)
	case 1:
		return json.Marshal(raw[0])
	default:
		return json.Marshal(append([]string{}, raw...))
	}
}

func (raw *SchemaType) UnmarshalJSON(data []byte) error {
	var out any

	err := json.Unmarshal(data, &out)
	if err != nil {
		return err
	}

	return raw.parseRawOutput(out)
}

func (raw SchemaType) MarshalYAML() ([]byte, error) {
	switch len(raw) {
	case 0:
		return yaml.Marshal(nil)
	case 1:
		return yaml.Marshal(raw[0])
	default:
		return yaml.Marshal(append([]string{}, raw...))
	}
}

func (raw *SchemaType) UnmarshalYAML(data []byte) error {
	var out any

	err := yaml.Unmarshal(data, &out)
	if err != nil {
		return err
	}

	return raw.parseRawOutput(out)
}

func (raw *SchemaType) parseRawOutput(out any) error {
	act, ok := out.([]any)
	if !ok {
		str, ok := out.(string)
		if !ok {
			return fmt.Errorf("%w: %T (must be string, or array of string)", ErrInvalidSchemaFormat, out)
		}

		act = []any{str}
	}

	(*raw) = make(SchemaType, len(act))
	for i, v := range act {
		(*raw)[i], ok = v.(string)
		if !ok {
			return fmt.Errorf("%w inside types array: %T (must be string)", ErrInvalidSchemaFormat, v)
		}
	}

	return nil
}

type SchemaAdditionalProperties struct {
	boolVal *bool
	schema  *RawSchema
}

func (raw SchemaAdditionalProperties) MarshalJSON() ([]byte, error) {
	if raw.schema != nil {
		return json.Marshal(raw.schema)
	}

	return json.Marshal(raw.boolVal)
}

func (raw *SchemaAdditionalProperties) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &raw.boolVal)
	if err != nil {
		err = json.Unmarshal(data, &raw.schema)
	}

	return err
}

func (raw SchemaAdditionalProperties) MarshalYAML() ([]byte, error) {
	if raw.schema != nil {
		return yaml.Marshal(raw.schema)
	}

	return yaml.Marshal(raw.boolVal)
}

func (raw *SchemaAdditionalProperties) UnmarshalYAML(data []byte) error {
	err := yaml.Unmarshal(data, &raw.boolVal)
	if err != nil {
		err = yaml.Unmarshal(data, &raw.schema)
	}

	return err
}

func AdditionalPropertiesBool(b bool) *SchemaAdditionalProperties {
	return &SchemaAdditionalProperties{boolVal: lo.ToPtr(b)}
}

func AdditionalPropertiesSchema(schema *RawSchema) *SchemaAdditionalProperties {
	return &SchemaAdditionalProperties{schema: schema}
}

// RawSchema represents our internal implementation of json schema, before being parsed.
type RawSchema struct {
	// Agora specific.

	NoGenerate bool `json:"noGenerate,omitempty" yaml:"noGenerate,omitempty"`

	// JSON Schema fields.
	// https://json-schema.org/understanding-json-schema/keywords

	Anchor        string               `json:"$anchor,omitempty"        yaml:"anchor,omitempty"`
	Comment       string               `json:"$comment,omitempty"       yaml:"comment,omitempty"`
	DynamicAnchor string               `json:"$dynamicAnchor,omitempty" yaml:"dynamicAnchor,omitempty"`
	Defs          map[string]RawSchema `json:"$defs,omitempty"          yaml:"defs,omitempty"`
	DynamicRef    string               `json:"$dynamicRef,omitempty"    yaml:"dynamicRef,omitempty"`
	ID            string               `json:"$id,omitempty"            yaml:"id,omitempty"`
	Ref           string               `json:"$ref,omitempty"           yaml:"ref,omitempty"`

	//nolint:lll
	AdditionalProperties *SchemaAdditionalProperties `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`

	AllOf             []RawSchema          `json:"allOf,omitempty"             yaml:"allOf,omitempty"`
	AnyOf             []RawSchema          `json:"anyOf,omitempty"             yaml:"anyOf,omitempty"`
	Const             *any                 `json:"const,omitempty"             yaml:"const,omitempty"`
	Contains          *RawSchema           `json:"contains,omitempty"          yaml:"contains,omitempty"`
	ContentEncoding   string               `json:"contentEncoding,omitempty"   yaml:"contentEncoding,omitempty"`
	ContentMediaType  string               `json:"contentMediaType,omitempty"  yaml:"contentMediaType,omitempty"`
	ContentSchema     *RawSchema           `json:"contentSchema,omitempty"     yaml:"contentSchema,omitempty"`
	Default           *any                 `json:"default,omitempty"           yaml:"default,omitempty"`
	DependentRequired map[string][]string  `json:"dependentRequired,omitempty" yaml:"dependentRequired,omitempty"`
	DependentSchemas  map[string]RawSchema `json:"dependentSchemas,omitempty"  yaml:"dependentSchemas,omitempty"`
	Description       string               `json:"description,omitempty"       yaml:"description,omitempty"`
	If                *RawSchema           `json:"if,omitempty"                yaml:"if,omitempty"`
	Then              *RawSchema           `json:"then,omitempty"              yaml:"then,omitempty"`
	Else              *RawSchema           `json:"else,omitempty"              yaml:"else,omitempty"`
	Enum              []any                `json:"enum,omitempty"              yaml:"enum,omitempty"`
	Examples          []any                `json:"examples,omitempty"          yaml:"examples,omitempty"`
	ExclusiveMaximum  *float64             `json:"exclusiveMaximum,omitempty"  yaml:"exclusiveMaximum,omitempty"`
	ExclusiveMinimum  *float64             `json:"exclusiveMinimum,omitempty"  yaml:"exclusiveMinimum,omitempty"`
	Format            string               `json:"format,omitempty"            yaml:"format,omitempty"`
	Items             *RawSchema           `json:"items,omitempty"             yaml:"items,omitempty"`
	MaxContains       *int                 `json:"maxContains,omitempty"       yaml:"maxContains,omitempty"`
	Maximum           *float64             `json:"maximum,omitempty"           yaml:"maximum,omitempty"`
	MaxItems          *int                 `json:"maxItems,omitempty"          yaml:"maxItems,omitempty"`
	MaxLength         *int                 `json:"maxLength,omitempty"         yaml:"maxLength,omitempty"`
	MaxProperties     *int                 `json:"maxProperties,omitempty"     yaml:"maxProperties,omitempty"`
	MinContains       *int                 `json:"minContains,omitempty"       yaml:"minContains,omitempty"`
	Minimum           *float64             `json:"minimum,omitempty"           yaml:"minimum,omitempty"`
	MinItems          *int                 `json:"minItems,omitempty"          yaml:"minItems,omitempty"`
	MinLength         *int                 `json:"minLength,omitempty"         yaml:"minLength,omitempty"`
	MinProperties     *int                 `json:"minProperties,omitempty"     yaml:"minProperties,omitempty"`
	MultipleOf        *float64             `json:"multipleOf,omitempty"        yaml:"multipleOf,omitempty"`
	Not               *RawSchema           `json:"not,omitempty"               yaml:"not,omitempty"`
	OneOf             []RawSchema          `json:"oneOf,omitempty"             yaml:"oneOf,omitempty"`
	Pattern           string               `json:"pattern,omitempty"           yaml:"pattern,omitempty"`
	PatternProperties map[string]RawSchema `json:"patternProperties,omitempty" yaml:"patternProperties,omitempty"`
	Properties        map[string]RawSchema `json:"properties,omitempty"        yaml:"properties,omitempty"`
	PropertyNames     *RawSchema           `json:"propertyNames,omitempty"     yaml:"propertyNames,omitempty"`
	Required          []string             `json:"required,omitempty"          yaml:"required,omitempty"`
	Title             string               `json:"title,omitempty"             yaml:"title,omitempty"`
	Type              SchemaType           `json:"type,omitempty"              yaml:"type,omitempty"`
}

func (schema RawSchema) Validate(flags ...SchemaValidateFlag) error {
	// Once a generation is disabled, it is for good.
	if schema.NoGenerate {
		flags = lo.Without(flags, SchemaValidateFlagGenerate)
	}

	if len(schema.Type) == 0 {
		return fmt.Errorf("%w: missing 'type'", ErrInvalidSchema)
	}

	// https://developers.openai.com/api/docs/guides/structured-outputs/#supported-schemas
	if lo.Contains(flags, SchemaValidateFlagGenerate) {
		allowedTypes := []string{
			"string",
			"number",
			"boolean",
			"integer",
			"object",
			"array",
		}

		const expectedSchemaTypeLen = 2

		// We already validated that the schema type is not empty, so index 0 will exist.
		if len(schema.Type) > expectedSchemaTypeLen {
			return fmt.Errorf(
				"%w: unsupported 'type' value for AI generation: '%s': a maximum of 2 values can be provided",
				ErrInvalidSchema, schema.Type,
			)
		}

		if len(schema.Type) == expectedSchemaTypeLen && !lo.Contains(schema.Type, "null") {
			return fmt.Errorf(
				"%w: unsupported 'type' value for AI generation: '%s': union is only allowed with null type",
				ErrInvalidSchema, schema.Type,
			)
		}

		inter := lo.Intersect(allowedTypes, schema.Type)
		if len(inter) != 1 {
			return fmt.Errorf(
				"%w: unsupported 'type' value for AI generation: '%s': value must be in %v",
				ErrInvalidSchema, schema.Type, allowedTypes,
			)
		}

		allowedFormats := []string{
			"date-time",
			"time",
			"date",
			"duration",
			"email",
			"hostname",
			"ipv4",
			"ipv6",
			"uuid",
		}

		if schema.Format != "" && !lo.Contains(allowedFormats, schema.Format) {
			return fmt.Errorf(
				"%w: unsupported 'format' value for AI generation: '%s': value must be in %v",
				ErrInvalidSchema, schema.Format, allowedFormats,
			)
		}
	}

	for path, s := range schema.schemaFields() {
		err := s.Validate(flags...)
		if err != nil {
			return fmt.Errorf("%w: %s: %w", ErrInvalidSchema, path, err)
		}
	}

	return nil
}

// schemaFields returns a map of all the nested schemas, with their JSON path.
func (schema RawSchema) schemaFields() map[string]*RawSchema {
	out := make(map[string]*RawSchema)

	for path, s := range schema.Defs {
		out["defs."+path] = &s
	}

	if schema.AdditionalProperties != nil && schema.AdditionalProperties.schema != nil {
		out["additionalProperties"] = schema.AdditionalProperties.schema
	}

	for i, s := range schema.AllOf {
		out[fmt.Sprintf("allOf[%d]", i)] = &s
	}

	for i, s := range schema.AnyOf {
		out[fmt.Sprintf("anyOf[%d]", i)] = &s
	}

	if schema.Contains != nil {
		out["contains"] = schema.Contains
	}

	if schema.ContentSchema != nil {
		out["contentSchema"] = schema.ContentSchema
	}

	for path, s := range schema.DependentSchemas {
		out["dependentSchemas."+path] = &s
	}

	if schema.If != nil {
		out["if"] = schema.If
	}

	if schema.Then != nil {
		out["then"] = schema.Then
	}

	if schema.Else != nil {
		out["else"] = schema.Else
	}

	if schema.Items != nil {
		out["items"] = schema.Items
	}

	if schema.Not != nil {
		out["not"] = schema.Not
	}

	for i, s := range schema.OneOf {
		out[fmt.Sprintf("oneOf[%d]", i)] = &s
	}

	for path, s := range schema.PatternProperties {
		out["patternProperties."+path] = &s
	}

	for path, s := range schema.Properties {
		out["properties."+path] = &s
	}

	if schema.PropertyNames != nil {
		out["propertyNames"] = schema.PropertyNames
	}

	return out
}

// Apply prepareSchema on a map of nested schemas.
func prepareSchemaMap(src map[string]RawSchema, flags ...SchemaBuildFlag) (map[string]RawSchema, error) {
	if len(src) == 0 {
		return nil, nil
	}

	out := make(map[string]RawSchema)

	for k, v := range src {
		err := prepareSchema(&v, flags...)
		if errors.Is(err, ErrSchemaNotGeneratable) {
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("prepare schema: %s: %w", k, err)
		}

		out[k] = v
	}

	return out, nil
}

// Apply prepareSchema on a list of nested schemas.
func prepareSchemaList(src []RawSchema, flags ...SchemaBuildFlag) ([]RawSchema, error) {
	if len(src) == 0 {
		return nil, nil
	}

	out := make([]RawSchema, 0, len(src))

	for _, v := range src {
		err := prepareSchema(&v, flags...)
		if errors.Is(err, ErrSchemaNotGeneratable) {
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("prepare schema: %w", err)
		}

		out = append(out, v)
	}

	return out, nil
}

// Prepare schema for build.
func prepareSchemaNested(target *RawSchema, flags ...SchemaBuildFlag) (*RawSchema, error) {
	if target == nil {
		return nil, nil
	}

	err := prepareSchema(target, flags...)
	if err != nil {
		return nil, fmt.Errorf("prepare schema: %w", err)
	}

	return target, nil
}

// Prepare schema for build.
func prepareSchema(src *RawSchema, flags ...SchemaBuildFlag) error {
	err := src.Validate(SchemaValidateFlagGenerate)
	if err != nil {
		return fmt.Errorf("validate schema: %w", err)
	}

	if lo.Contains(flags, SchemaBuildFlagAI) {
		if src.NoGenerate {
			return ErrSchemaNotGeneratable
		}

		// Save properties before overwriting src, so the loop below can filter
		// NoGenerate entries from the original set.
		originalProperties := src.Properties

		// Restrict to OpenAI-supported fields. Assigned in-place so that the
		// caller's pointer (and BuildSchema's json.Marshal target) reflects the
		// change. additionalProperties must be false per the OpenAI structured
		// outputs spec.
		*src = RawSchema{
			Type:                 src.Type,
			AdditionalProperties: &SchemaAdditionalProperties{boolVal: lo.ToPtr(false)},
			Pattern:              src.Pattern,
			Enum:                 src.Enum,
			AnyOf:                src.AnyOf,
			Format:               src.Format,
			MultipleOf:           src.MultipleOf,
			Maximum:              src.Maximum,
			ExclusiveMaximum:     src.ExclusiveMaximum,
			Minimum:              src.Minimum,
			ExclusiveMinimum:     src.ExclusiveMinimum,
			MinItems:             src.MinItems,
			MaxItems:             src.MaxItems,
			MinLength:            src.MinLength,
			MaxLength:            src.MaxLength,
			Description:          src.Description,
			Items:                src.Items,
			Examples:             src.Examples,
		}

		generatableProperties := make(map[string]RawSchema)

		for k, v := range originalProperties {
			if v.NoGenerate {
				continue
			}

			generatableProperties[k] = v
		}

		src.Properties = generatableProperties
		src.Required = lo.Keys(generatableProperties)
	}

	if lo.Contains(flags, SchemaBuildFlagPartial) {
		// Delete validation properties.
		src.Required = nil
		src.MinLength = nil
		src.MinItems = nil
		src.Minimum = nil
		src.ExclusiveMinimum = nil
	}

	src.Defs, err = prepareSchemaMap(src.Defs, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: defs: %w", err)
	}

	if src.AdditionalProperties != nil {
		src.AdditionalProperties.schema, err = prepareSchemaNested(src.AdditionalProperties.schema, flags...)
		if err != nil {
			return fmt.Errorf("prepare schema: additionalProperties: %w", err)
		}
	}

	src.AllOf, err = prepareSchemaList(src.AllOf, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: allOf: %w", err)
	}

	src.AnyOf, err = prepareSchemaList(src.AnyOf, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: anyOf: %w", err)
	}

	src.Contains, err = prepareSchemaNested(src.Contains, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: contains: %w", err)
	}

	src.ContentSchema, err = prepareSchemaNested(src.ContentSchema, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: contentSchema: %w", err)
	}

	src.DependentSchemas, err = prepareSchemaMap(src.DependentSchemas, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: dependentSchemas: %w", err)
	}

	src.If, err = prepareSchemaNested(src.If, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: if: %w", err)
	}

	src.Then, err = prepareSchemaNested(src.Then, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: then: %w", err)
	}

	src.Else, err = prepareSchemaNested(src.Else, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: else: %w", err)
	}

	src.Items, err = prepareSchemaNested(src.Items, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: items: %w", err)
	}

	src.Not, err = prepareSchemaNested(src.Not, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: not: %w", err)
	}

	src.OneOf, err = prepareSchemaList(src.OneOf, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: oneOf: %w", err)
	}

	src.PatternProperties, err = prepareSchemaMap(src.PatternProperties, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: patternProperties: %w", err)
	}

	src.Properties, err = prepareSchemaMap(src.Properties, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: properties: %w", err)
	}

	src.PropertyNames, err = prepareSchemaNested(src.PropertyNames, flags...)
	if err != nil {
		return fmt.Errorf("prepare schema: propertyNames: %w", err)
	}

	return nil
}

// BuildSchema builds a JSON schema from a raw schema.
func BuildSchema(src *RawSchema, flags ...SchemaBuildFlag) (*jsonschema.Resolved, error) {
	err := prepareSchema(src, flags...)
	if err != nil {
		return nil, err
	}

	mrsh, err := json.Marshal(src)
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}

	var schema jsonschema.Schema

	err = json.Unmarshal(mrsh, &schema)
	if err != nil {
		return nil, fmt.Errorf("unmarshal schema: %w", err)
	}

	resolved, err := schema.Resolve(nil)
	if err != nil {
		return nil, fmt.Errorf("resolve schema: %w", err)
	}

	return resolved, nil
}
