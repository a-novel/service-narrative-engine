package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

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

// The rules a [FieldError] names, beside the tags of the validator.
const (
	// ruleMax refuses a list that holds more items, or a text more characters, than its cap.
	ruleMax = "max"
	// ruleMaxBytes refuses a text or a document over its cap, in bytes.
	ruleMaxBytes = "maxbytes"
	// ruleMaxDepth refuses a document that nests its arrays and objects deeper than its cap.
	ruleMaxDepth = "maxdepth"
	// ruleJSON refuses a document that is not JSON.
	ruleJSON = "json"
	// ruleNoNul refuses a document whose strings hold the NUL character.
	ruleNoNul = "nonul"
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

// nulEscape is how a JSON string spells the NUL character, past the backslash.
var nulEscape = []byte("u0000")

// documentShape reads a JSON document once. It returns how deep the document nests its arrays and objects,
// and whether one of its strings holds the NUL character, which PostgreSQL stores in no jsonb value.
//
// It follows the depth with a counter and no stack, since JSON closes what it opens. A text that is not JSON
// yields numbers that mean nothing, and never a fault.
func documentShape(document []byte) (int, bool) {
	var (
		depth, deepest int
		inString, nul  bool
	)

	for index := 0; index < len(document); index++ {
		switch char := document[index]; {
		case inString && char == '\\':
			// An escape spans the next byte, which may be a quote. No raw NUL passes for JSON, so the escape
			// is its only spelling.
			nul = nul || bytes.HasPrefix(document[index+1:], nulEscape)
			index++
		case char == '"':
			inString = !inString
		case inString:
		case char == '{' || char == '[':
			depth++
			deepest = max(deepest, depth)
		case char == '}' || char == ']':
			depth--
		}
	}

	return deepest, nul
}

// checkDocument returns a [FieldError] at path unless document is JSON of bytesMax bytes at most, nested
// depthMax levels at most, whose strings hold no NUL character. It reads nothing else of the document: what
// the document means is the client's to check.
func checkDocument(path string, document json.RawMessage, bytesMax, depthMax int) error {
	if len(document) > bytesMax {
		return &FieldError{Path: path, Rule: ruleMaxBytes}
	}

	// The depth comes first: the standard library refuses a document nested past its own limit as if it were
	// not JSON.
	depth, nul := documentShape(document)

	switch {
	case depth > depthMax:
		return &FieldError{Path: path, Rule: ruleMaxDepth}
	case !json.Valid(document):
		return &FieldError{Path: path, Rule: ruleJSON}
	case nul:
		return &FieldError{Path: path, Rule: ruleNoNul}
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
	}

	for tag, fn := range validations {
		err := validate.RegisterValidation(tag, fn)
		if err != nil {
			panic(err)
		}
	}
}
