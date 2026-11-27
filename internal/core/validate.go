package core

import (
	"cmp"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
	"golang.org/x/text/unicode/norm"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// ErrInvalidRequest is returned by a service Exec when its request fails struct validation.
var ErrInvalidRequest = errors.New("invalid request")

// inSet returns a validation that reports whether a string is one of set.
func inSet[T ~string](set []T) validator.Func {
	return func(fl validator.FieldLevel) bool {
		return slices.Contains(set, T(fl.Field().String()))
	}
}

// validateProjectLabel reports whether a genre of a project holds no whitespace at either end and no NUL, which
// the database driver drops on write. The package registers it as the "projectlabel" struct tag.
func validateProjectLabel(fl validator.FieldLevel) bool {
	label := fl.Field().String()

	return strings.TrimSpace(label) == label && !strings.ContainsRune(label, 0)
}

// projectTagRune lists what a word of a project tag starts with: a lowercase, modifier or caseless letter, or a digit.
const projectTagRune = `\p{Ll}\p{Lm}\p{Lo}\p{Nd}`

// projectTagWord matches a word of a project tag: a letter or digit, then the marks and zero-width joiners inside it,
// ending on a letter, a digit or a mark.
const projectTagWord = `[` + projectTagRune + `]` +
	`(?:[` + projectTagRune + `\p{M}\x{200C}\x{200D}]*[` + projectTagRune + `\p{M}])?`

// projectTagPattern matches a tag of a project in lower kebab case: words joined by single hyphens.
var projectTagPattern = regexp.MustCompile(`^` + projectTagWord + `(?:-` + projectTagWord + `)*$`)

// validateProjectTag reports whether a tag of a project is in lower kebab case, as "années-50" is: words from any
// script joined by single hyphens. A word starts with a lowercase, modifier or caseless letter or a digit, and holds
// the marks and zero-width joiners its script writes inside words, as Thai, Hindi or Persian do, so a mark or joiner
// left alone, as an emoji's variation selector is, makes no tag. The tag must be precomposed (NFC), so an accented
// letter has a single spelling. The package registers it as the "projecttag" struct tag.
func validateProjectTag(fl validator.FieldLevel) bool {
	tag := fl.Field().String()

	return projectTagPattern.MatchString(tag) && norm.NFC.IsNormalString(tag)
}

// logoChecksumPattern matches the checksum of a logo: a SHA-256 in lowercase hex.
var logoChecksumPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// validateLogoChecksum reports whether a string is the checksum of a logo: a SHA-256 in lowercase hex, as a project
// carries it. The package registers it as the "logochecksum" struct tag.
func validateLogoChecksum(fl validator.FieldLevel) bool {
	return logoChecksumPattern.MatchString(fl.Field().String())
}

// validateInstant reports whether a string is an RFC 3339 instant PostgreSQL accepts: a year from 1, at an
// offset under 16 hours. The package registers it as the "instant" struct tag.
func validateInstant(fl validator.FieldLevel) bool {
	instant, err := time.Parse(time.RFC3339, fl.Field().String())
	_, offset := instant.Zone()

	return err == nil && instant.Year() >= 1 && offset > -16*60*60 && offset < 16*60*60
}

// validatePluginVersion reports whether a string is a version the registry can hold, as [parsePluginVersion]
// reads it. The package registers it as the "pluginversion" struct tag.
func validatePluginVersion(fl validator.FieldLevel) bool {
	// FIXME: return the reason once a validation can. A validator.Func only answers yes or no, so what
	// parsePluginVersion says of a refused version is dropped here (go-playground/validator#1278).
	_, err := parsePluginVersion(fl.Field().String())

	return err == nil
}

// pluginSchemaKeywords lists the JSON Schema keywords whose value is a subschema, or a list of subschemas.
// additionalItems is older than draft 2020-12, and validators still apply it.
var pluginSchemaKeywords = []string{
	"additionalItems", "additionalProperties", "allOf", "anyOf", "contains", "contentSchema", "else", "if",
	"items", "not", "oneOf", "prefixItems", "propertyNames", "then", "unevaluatedItems", "unevaluatedProperties",
}

// pluginSchemaMapKeywords lists the JSON Schema keywords whose value maps names to subschemas. definitions
// and dependencies are older than draft 2020-12, and validators still apply them.
var pluginSchemaMapKeywords = []string{
	"$defs", "definitions", "dependencies", "dependentSchemas", "patternProperties", "properties",
}

// pluginSchemaRefused lists the JSON Schema keywords a plugin's schema cannot use: some validators lack the
// first two, and validators disagree on formats.
var pluginSchemaRefused = []string{"$dynamicRef", "$dynamicAnchor", "format"}

// validatePluginSchema reports whether a JSON document is a schema every host of a plugin reads alike: a JSON
// object whose subschemas only reference the schema itself, with a "$ref" that starts with "#", and use no
// keyword of [pluginSchemaRefused]. The package registers it as the "pluginschema" struct tag.
//
// It reads the subschemas by keyword, so a property named after a keyword, or the content of a default, is
// never read as a schema. It does not compile the schema: this validator reads a pattern as RE2, where a host
// reads it as ECMAScript.
func validatePluginSchema(fl validator.FieldLevel) bool {
	var schema map[string]any

	err := json.Unmarshal(fl.Field().Bytes(), &schema)
	if err != nil || schema == nil {
		return false
	}

	// The depth of a schema comes from the request, so the subschemas wait on a stack, with no recursion.
	pending := []map[string]any{schema}

	for len(pending) > 0 {
		subschema := pending[len(pending)-1]
		pending = pending[:len(pending)-1]

		if !pluginSubschemaAllowed(subschema) {
			return false
		}

		pending = appendPluginSubschemas(pending, subschema)
	}

	return true
}

// pluginSubschemaAllowed reports whether one subschema of a plugin's schema, its own subschemas aside, keeps
// to what [validatePluginSchema] allows.
func pluginSubschemaAllowed(subschema map[string]any) bool {
	if ref, ok := subschema["$ref"]; ok {
		target, isString := ref.(string)
		if !isString || !strings.HasPrefix(target, "#") {
			return false
		}
	}

	for _, keyword := range pluginSchemaRefused {
		if _, ok := subschema[keyword]; ok {
			return false
		}
	}

	return true
}

// appendPluginSubschemas appends to pending the subschemas a schema holds under its own keywords. A boolean
// subschema is left out: it holds no keyword.
func appendPluginSubschemas(pending []map[string]any, schema map[string]any) []map[string]any {
	for _, keyword := range pluginSchemaKeywords {
		switch value := schema[keyword].(type) {
		case map[string]any:
			pending = append(pending, value)
		case []any:
			pending = appendJSONObjects(pending, slices.Values(value))
		}
	}

	for _, keyword := range pluginSchemaMapKeywords {
		if children, ok := schema[keyword].(map[string]any); ok {
			pending = appendJSONObjects(pending, maps.Values(children))
		}
	}

	return pending
}

// appendJSONObjects appends to objects each of values that is a JSON object.
func appendJSONObjects(objects []map[string]any, values iter.Seq[any]) []map[string]any {
	for value := range values {
		if object, ok := value.(map[string]any); ok {
			objects = append(objects, object)
		}
	}

	return objects
}

// pluginExportNamePattern matches the name of an export. A link addresses an item as "<export>/<id>", so a
// name holds no slash.
var pluginExportNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

// pluginExportSelectPattern matches the path to the items of an export in the content: "$", then member
// names and "[*]", a subset of JSONPath (RFC 9535).
var pluginExportSelectPattern = regexp.MustCompile(`^\$(\.[A-Za-z_][A-Za-z0-9_]*|\[\*\])*$`)

// pluginExportPointerPattern matches a JSON Pointer (RFC 6901) into an item of an export.
var pluginExportPointerPattern = regexp.MustCompile(`^(/([^/~]|~[01])*)*$`)

// pluginExportMembers maps each member of an export, one named list of linkable items, to the pattern its
// value follows. An export holds every one of them as a string, and no other member.
var pluginExportMembers = map[string]*regexp.Regexp{
	// The translation key of the export's display name, which follows no pattern.
	"name": nil,
	// The path to the exported items.
	"select": pluginExportSelectPattern,
	// The pointer to the identifier inside each item.
	"id": pluginExportPointerPattern,
	// The pointer to the label inside each item.
	"label": pluginExportPointerPattern,
}

// validatePluginExports reports whether a JSON document holds the exports of a plugin, the lists of linkable
// items an instance exposes: a JSON object that maps each name [pluginExportNamePattern] matches to an export
// [pluginExportAllowed] accepts. The package registers it as the "pluginexports" struct tag.
func validatePluginExports(fl validator.FieldLevel) bool {
	var exports map[string]any

	err := json.Unmarshal(fl.Field().Bytes(), &exports)
	if err != nil || exports == nil {
		return false
	}

	for name, export := range exports {
		if !pluginExportNamePattern.MatchString(name) || !pluginExportAllowed(export) {
			return false
		}
	}

	return true
}

// pluginExportAllowed reports whether a JSON value is one export of a plugin: an object that holds each of
// [pluginExportMembers] as a string its pattern matches, and no other member.
func pluginExportAllowed(value any) bool {
	export, ok := value.(map[string]any)
	if !ok || len(export) != len(pluginExportMembers) {
		return false
	}

	for member, pattern := range pluginExportMembers {
		text, isString := export[member].(string)
		if !isString || (pattern != nil && !pattern.MatchString(text)) {
			return false
		}
	}

	return true
}

// validatePluginEnglish reports whether the translations of a plugin, a map by language, hold
// [PluginLanguageEnglish]. The package registers it as the "pluginenglish" struct tag.
func validatePluginEnglish(fl validator.FieldLevel) bool {
	return fl.Field().MapIndex(reflect.ValueOf(PluginLanguageEnglish)).IsValid()
}

// validateJSONObject reports whether a JSON document is an object. The package registers it as the
// "jsonobject" struct tag.
func validateJSONObject(fl validator.FieldLevel) bool {
	var object map[string]json.RawMessage

	err := json.Unmarshal(fl.Field().Bytes(), &object)

	return err == nil && object != nil
}

// validateNoNul reports whether a text holds no NUL character, which PostgreSQL stores in no text. The package
// registers it as the "nonul" struct tag, the rule a document breaks for the same character.
func validateNoNul(fl validator.FieldLevel) bool {
	return !strings.ContainsRune(fl.Field().String(), 0)
}

// The rules a [FieldError] names, beside the tags of the validator.
const (
	// ruleMax refuses a list that holds more items, or a text more characters, than its cap.
	ruleMax = "max"
	// ruleMaxBytes refuses a text or a document over its cap, in bytes. A document counts each of its numbers
	// as the database stores it: written in full, without an exponent.
	ruleMaxBytes = "maxbytes"
	// ruleMaxDepth refuses a document that nests its arrays and objects deeper than its cap.
	ruleMaxDepth = "maxdepth"
	// ruleMaxDigits refuses a document that holds a number of more than [documentDigitsMax] digits, written
	// in full.
	ruleMaxDigits = "maxdigits"
	// ruleJSON refuses a document that is not JSON in UTF-8.
	ruleJSON = "json"
	// ruleNoNul refuses a document whose strings hold the NUL character.
	ruleNoNul = "nonul"
	// ruleNoLoneSurrogate refuses a document whose strings hold one half of a surrogate pair without the other.
	ruleNoLoneSurrogate = "nolonesurrogate"
)

// FieldError is returned when a field of a request breaks a rule the validator's tags cannot hold: a cap the
// configuration sets, or a check over a whole document. It wraps [ErrInvalidRequest].
type FieldError struct {
	// Path locates the field in the request, in lower camel case, as "instances[2].content" does.
	Path string
	// Rule names the rule the field broke, as a validator tag names its own.
	Rule string
}

func (err *FieldError) Error() string {
	return err.Path + " breaks the " + err.Rule + " rule"
}

func (err *FieldError) Unwrap() error {
	return ErrInvalidRequest
}

// How many bytes an escape of a JSON string spans, from its backslash.
const (
	// plainEscapeWidth spans the backslash and one byte, which may be a quote.
	plainEscapeWidth = len(`\"`)
	// unitEscapeWidth spans a \u escape, which spells one UTF-16 code unit in four hex digits.
	unitEscapeWidth = len(`\u0000`)
	// pairEscapeWidth spans a surrogate pair: two \u escapes, the high half then the low one.
	pairEscapeWidth = len(`\ud83d\ude00`)
)

// escapedUnit returns the UTF-16 code unit a \u escape spells, and false unless text starts with one.
func escapedUnit(text []byte) (rune, bool) {
	if len(text) < unitEscapeWidth || text[0] != '\\' || text[1] != 'u' {
		return 0, false
	}

	unit, err := strconv.ParseUint(string(text[len(`\u`):unitEscapeWidth]), 16, 16)

	return rune(unit), err == nil
}

// readEscape reads the escape text starts with. It returns how many bytes the escape spans, and the rule it
// breaks, if any.
//
// A \u escape spells one UTF-16 code unit, and JSON takes any (RFC 8259, section 7). PostgreSQL stores neither
// NUL nor half of a surrogate pair in a jsonb value, and an escape is the only spelling of either: a raw NUL
// is no JSON, and a raw surrogate no UTF-8.
func readEscape(text []byte) (int, string) {
	unit, ok := escapedUnit(text)

	switch {
	case !ok:
		return plainEscapeWidth, ""
	case unit == 0:
		return unitEscapeWidth, ruleNoNul
	case !utf16.IsSurrogate(unit):
		return unitEscapeWidth, ""
	}

	low, _ := escapedUnit(text[unitEscapeWidth:])
	if utf16.DecodeRune(unit, low) == unicode.ReplacementChar {
		return unitEscapeWidth, ruleNoLoneSurrogate
	}

	return pairEscapeWidth, ""
}

// documentDigitsMax caps the digits of a number of a document, written in full. PostgreSQL stores a number of
// a jsonb value without its exponent, as a numeric: 16383 digits at most past the decimal point, and more
// before it.
//
// The cap counts every digit of the number, on either side of the point, so it also refuses the few longer
// numbers PostgreSQL would store. Every number an IEEE 754 double holds passes.
const documentDigitsMax = 16383

// decimal is the base a JSON number is written in.
const decimal = 10

// readNumber reads the number text starts with. It returns how many bytes the number spans, how many digits
// it holds, and its exponent without its sign: how many more digits the number takes written in full. JSON
// bounds neither (RFC 8259, section 6).
func readNumber(text []byte) (int, int, int) {
	var (
		width, digits, exponent int
		inExponent              bool
	)

	for ; width < len(text); width++ {
		switch char := text[width]; {
		case char == 'e' || char == 'E':
			inExponent = true
		case char == '+' || char == '-' || char == '.':
		case char < '0' || char > '9':
			return width, digits, exponent
		case !inExponent:
			digits++
		// Past the cap the exponent stops growing, so none overflows the count.
		case exponent <= documentDigitsMax:
			exponent = exponent*decimal + int(char-'0')
		}
	}

	return width, digits, exponent
}

// documentShape is what one reading of a JSON document finds.
type documentShape struct {
	// depth is how deep the document nests its arrays and objects.
	depth int
	// grown is how many bytes the document takes once stored, past its own: the database writes each number
	// in full, and an exponent adds that many digits at most.
	grown int
	// broken is the first rule a string or a number of the document breaks, if any.
	broken string
}

// readDocument reads a JSON document once, for what the standard library does not check and PostgreSQL does.
//
// It follows the depth with a counter and no stack, since JSON closes what it opens. A text that is not JSON
// yields values that mean nothing, and never a fault.
func readDocument(document []byte) documentShape {
	var (
		shape    documentShape
		depth    int
		inString bool
	)

	for index := 0; index < len(document); index++ {
		switch char := document[index]; {
		case inString && char == '\\':
			width, rule := readEscape(document[index:])
			shape.broken = cmp.Or(shape.broken, rule)
			index += width - 1
		case char == '"':
			inString = !inString
		case inString:
		case char == '{' || char == '[':
			depth++
			shape.depth = max(shape.depth, depth)
		case char == '}' || char == ']':
			depth--
		case char == '-' || char >= '0' && char <= '9':
			width, digits, exponent := readNumber(document[index:])
			if digits+exponent > documentDigitsMax {
				shape.broken = cmp.Or(shape.broken, ruleMaxDigits)
			}

			shape.grown += exponent
			index += width - 1
		}
	}

	return shape
}

// checkDocument returns a [FieldError] at path unless document is JSON in UTF-8 (RFC 8259, section 8.1) that
// PostgreSQL stores, within the caps: nested depthMax levels at most, and of bytesMax bytes at most with its
// numbers written in full, as the database stores them. PostgreSQL stores no NUL character, no half of a
// surrogate pair, and no number of more than [documentDigitsMax] digits. The check reads nothing else of the
// document: what the document means is the client's to check.
func checkDocument(path string, document json.RawMessage, bytesMax, depthMax int) error {
	if len(document) > bytesMax {
		return &FieldError{Path: path, Rule: ruleMaxBytes}
	}

	// The depth comes first: the standard library refuses a document nested past its own limit as if it were
	// not JSON.
	shape := readDocument(document)

	switch {
	case shape.depth > depthMax:
		return &FieldError{Path: path, Rule: ruleMaxDepth}
	// The standard library takes any bytes inside a string, and the database driver would replace the ones
	// that are not UTF-8.
	case !json.Valid(document) || !utf8.Valid(document):
		return &FieldError{Path: path, Rule: ruleJSON}
	case shape.broken != "":
		return &FieldError{Path: path, Rule: shape.broken}
	case len(document)+shape.grown > bytesMax:
		return &FieldError{Path: path, Rule: ruleMaxBytes}
	}

	return nil
}

func init() {
	validations := map[string]validator.Func{
		"notblank":      validators.NotBlank,
		"language":      inSet(ProjectLanguages),
		"projectlabel":  validateProjectLabel,
		"projecttag":    validateProjectTag,
		"logochecksum":  validateLogoChecksum,
		"instant":       validateInstant,
		"pluginversion": validatePluginVersion,
		"pluginmodule":  inSet([]PluginModuleKind{PluginModuleUI, PluginModuleMigration}),
		"pluginschema":  validatePluginSchema,
		"pluginexports": validatePluginExports,
		"pluginenglish": validatePluginEnglish,
		"jsonobject":    validateJSONObject,
		ruleNoNul:       validateNoNul,
	}

	for tag, fn := range validations {
		err := validate.RegisterValidation(tag, fn)
		if err != nil {
			panic(err)
		}
	}
}
