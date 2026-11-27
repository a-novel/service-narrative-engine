package core_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
	"github.com/a-novel/service-json-keys/v2/pkg/go/servicejsonkeystest"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/v2/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// pluginExportsCases holds the documents the exports of a version accept or refuse. The JS client's tests run
// the same file, so both validators agree on every case.
//
//go:embed testdata/plugin.exports.cases.json
var pluginExportsCases []byte

// The caps of a published version: 1 MiB for the UI module, 256 KiB for each other document, and 4096
// characters for the proof.
const (
	pluginUIBytesMax       = 1 << 20
	pluginDocumentBytesMax = 256 << 10
	pluginProofMax         = 4096
)

// pluginSchemaOf returns a plugin schema that holds a subschema under each keyword of a JSON Schema that takes
// one, a list of them, or a map of them.
func pluginSchemaOf(subschema string) map[string]string {
	single := []string{
		"additionalItems", "additionalProperties", "contains", "contentSchema", "else", "if", "items", "not",
		"propertyNames", "then", "unevaluatedItems", "unevaluatedProperties",
	}
	list := []string{"allOf", "anyOf", "oneOf", "prefixItems"}
	named := []string{"$defs", "definitions", "dependencies", "dependentSchemas", "patternProperties", "properties"}

	schemas := make(map[string]string, len(single)+len(list)+len(named))

	for _, keyword := range single {
		schemas[keyword] = `{"type": "object", "` + keyword + `": ` + subschema + `}`
	}

	// A list and a map hold a boolean subschema before the one under test.
	for _, keyword := range list {
		schemas[keyword] = `{"type": "object", "` + keyword + `": [true, ` + subschema + `]}`
	}

	for _, keyword := range named {
		schemas[keyword] = `{"type": "object", "` + keyword + `": {"a": false, "b": ` + subschema + `}}`
	}

	return schemas
}

func TestPluginVersionPublish(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	const (
		schema  = `{"type": "object", "default": {"beats": []}, "properties": {"beats": {"type": "array"}}}`
		exports = `{"scenes": {"name": "exports.scenes", "select": "$.beats[*].scenes[*]", "id": "/id", "label": "/title"}}`
		ui      = `export default { mount() {} };`
		migrate = `export default (content) => ({ migrated: content });`

		// The proof a publisher sends with a version, and the ID of the API key that signed both the proof
		// and the request.
		proof    = "header.claims.signature"
		apiKeyID = "publisher-key"
	)

	english := core.PluginVersionPublishLanguage{
		Title: "Scenes Composer", Description: "Lays a story out in beats and scenes.",
		Messages: json.RawMessage(`{"exports": {"scenes": "Scenes"}}`),
	}
	french := core.PluginVersionPublishLanguage{
		Title: "Compositeur de scènes", Description: "Découpe une histoire en temps forts et en scènes.",
		Messages: json.RawMessage(`{"exports": {"scenes": "Scènes"}}`),
	}

	// valid returns a request the registry takes, changed by each of changes.
	valid := func(changes ...func(request *core.PluginVersionPublishRequest)) *core.PluginVersionPublishRequest {
		request := &core.PluginVersionPublishRequest{
			Plugin: "scenes-composer", Version: "1.10.0",
			Schema: json.RawMessage(schema), Exports: json.RawMessage(exports),
			UI: []byte(ui), Migration: []byte(migrate),
			Languages: map[string]core.PluginVersionPublishLanguage{"fr": french, "en": english},
		}

		for _, change := range changes {
			change(request)
		}

		return request
	}

	// signed gives a request the proof of its version, and the API key that authenticated it.
	signed := func(request *core.PluginVersionPublishRequest) { request.Proof, request.APIKeyID = proof, apiKeyID }

	noMigration := func(request *core.PluginVersionPublishRequest) { request.Migration = nil }

	// digest returns the SHA-256 of a module in lowercase hex, as a proof carries it.
	digest := func(module string) string {
		sum := sha256.Sum256([]byte(module))

		return hex.EncodeToString(sum[:])
	}

	// proofClaims returns the claims the proof of a valid request adds to the registered ones, changed by each
	// of changes.
	proofClaims := func(changes ...func(claims map[string]any)) map[string]any {
		claims := map[string]any{
			"plugin": "scenes-composer", "version": "1.10.0",
			"uiSHA256": digest(ui), "migrationSHA256": digest(migrate),
		}

		for _, change := range changes {
			change(claims)
		}

		return claims
	}

	withClaim := func(claim string, value any) func(claims map[string]any) {
		return func(claims map[string]any) { claims[claim] = value }
	}

	noMigrationClaim := func(claims map[string]any) { delete(claims, "migrationSHA256") }

	daoLanguage := func(language string, translations core.PluginVersionPublishLanguage) dao.PluginVersionInsertLanguage {
		return dao.PluginVersionInsertLanguage{
			Language: language, Title: translations.Title, Description: translations.Description,
			Messages: translations.Messages,
		}
	}

	// stored returns the write a valid request asks for, changed by each of changes. English comes first: the
	// languages are stored in the order of their tags.
	stored := func(changes ...func(request *dao.PluginVersionInsertRequest)) *dao.PluginVersionInsertRequest {
		request := &dao.PluginVersionInsertRequest{
			Plugin: "scenes-composer", Major: 1, Minor: 10,
			Schema: json.RawMessage(schema), Exports: json.RawMessage(exports),
			UI: []byte(ui), Migration: []byte(migrate),
			Languages: []dao.PluginVersionInsertLanguage{daoLanguage("en", english), daoLanguage("fr", french)},
		}

		for _, change := range changes {
			change(request)
		}

		return request
	}

	// stores gives a write the proof of a signed request, and the ID of its key.
	stores := func(request *dao.PluginVersionInsertRequest) {
		request.Proof, request.APIKeyID = new(proof), new(apiKeyID)
	}

	// jsonKeysMock is what JSON Keys answers to the proof of a request.
	type jsonKeysMock struct {
		// keyID is the ID of the API key that signed the proof.
		keyID string
		// claims are the claims JSON Keys decodes from the proof.
		claims map[string]any

		err error
	}

	type testCase struct {
		name string

		request *core.PluginVersionPublishRequest

		// jsonKeysMock is the answer of JSON Keys, which verifies the proof of the request. It is not called
		// when the mock is nil.
		jsonKeysMock *jsonKeysMock

		// daoRequest is the write the service asks for, whose instant the service captures. The DAO is not
		// called when it is nil, so nothing is stored.
		daoRequest *dao.PluginVersionInsertRequest
		daoErr     error

		expectErr error
		// expectFault marks an error the request did not cause, which never reads as an invalid request.
		expectFault bool
		// expectField is the field a refused request breaks, as the validator names it under the request, and
		// expectRule the rule it breaks.
		expectField string
		expectRule  string
		// expectDocument is the document the database would not store as it is, by its path in the body, and
		// the rule it breaks.
		expectDocument *core.FieldError
	}

	// refusedProof returns a case for a signed request, changed by each of changes, whose proof does not vouch
	// for its version. Nothing is stored.
	refusedProof := func(name string, answer *jsonKeysMock, changes ...func(*core.PluginVersionPublishRequest)) testCase {
		changes = append([]func(*core.PluginVersionPublishRequest){signed}, changes...)

		return testCase{
			name: "Error/Proof/" + name, request: valid(changes...),
			jsonKeysMock: answer, expectErr: core.ErrPluginProof,
		}
	}

	// invalid returns a case for a request a check refuses, naming the field and the rule it breaks.
	invalid := func(name, field, rule string, change func(request *core.PluginVersionPublishRequest)) testCase {
		return testCase{
			name: "Error/" + name, request: valid(change),
			expectErr: core.ErrInvalidRequest, expectField: field, expectRule: rule,
		}
	}

	// unstorable returns a case for a request whose document the database would not store as it is, naming
	// the document by its path in the body and the rule it breaks.
	unstorable := func(name, path, rule string, change func(request *core.PluginVersionPublishRequest)) testCase {
		return testCase{
			name: "Error/" + name, request: valid(change),
			expectErr: core.ErrInvalidRequest, expectDocument: &core.FieldError{Path: path, Rule: rule},
		}
	}

	withSchema := func(schema string) func(request *core.PluginVersionPublishRequest) {
		return func(request *core.PluginVersionPublishRequest) { request.Schema = json.RawMessage(schema) }
	}

	withEnglish := func(change func(english *core.PluginVersionPublishLanguage)) func(*core.PluginVersionPublishRequest) {
		return func(request *core.PluginVersionPublishRequest) {
			changed := english
			change(&changed)

			request.Languages = map[string]core.PluginVersionPublishLanguage{"en": changed}
		}
	}

	// A document of exactly size bytes: a JSON object whose one member pads it.
	document := func(size int) json.RawMessage {
		return json.RawMessage(`{"pad": "` + strings.Repeat("a", size-len(`{"pad": ""}`)) + `"}`)
	}

	// A schema whose items nest depth times, around a last subschema.
	nested := func(depth int, last string) string {
		return strings.Repeat(`{"items": `, depth) + last + strings.Repeat(`}`, depth)
	}

	manyLanguages := func(count int) func(request *core.PluginVersionPublishRequest) {
		return func(request *core.PluginVersionPublishRequest) {
			request.Languages = map[string]core.PluginVersionPublishLanguage{"en": english}

			// A private use tag per language: x-a0, x-a1, and so on.
			for i := range count - 1 {
				request.Languages["x-a"+strconv.Itoa(i)] = english
			}
		}
	}

	testCases := []testCase{
		{
			name: "Success",

			request: valid(),

			daoRequest: stored(),
		},
		{
			name: "Success/NoMigration",

			request: valid(noMigration),

			daoRequest: stored(func(request *dao.PluginVersionInsertRequest) { request.Migration = nil }),
		},
		{
			// The proof is stored with the ID of the key that signed it.
			name: "Success/Proof",

			request: valid(signed),

			jsonKeysMock: &jsonKeysMock{keyID: apiKeyID, claims: proofClaims()},

			daoRequest: stored(stores),
		},
		{
			// The proof of a version without a migration carries no digest of one.
			name: "Success/Proof/NoMigration",

			request: valid(signed, noMigration),

			jsonKeysMock: &jsonKeysMock{keyID: apiKeyID, claims: proofClaims(noMigrationClaim)},

			daoRequest: stored(stores, func(request *dao.PluginVersionInsertRequest) { request.Migration = nil }),
		},
		{
			name: "Success/Proof/Longest",

			request: valid(signed, func(request *core.PluginVersionPublishRequest) {
				request.Proof = strings.Repeat("a", pluginProofMax)
			}),

			jsonKeysMock: &jsonKeysMock{keyID: apiKeyID, claims: proofClaims()},

			daoRequest: stored(stores, func(request *dao.PluginVersionInsertRequest) {
				request.Proof = new(strings.Repeat("a", pluginProofMax))
			}),
		},
		{
			// A prerelease publishes as a release does.
			name: "Success/Prerelease",

			request: valid(func(request *core.PluginVersionPublishRequest) { request.Version = "2.0.0-dev.1" }),

			daoRequest: stored(func(request *dao.PluginVersionInsertRequest) {
				request.PluginVersionKey = dao.PluginVersionKey{Plugin: "scenes-composer", Major: 2, Prerelease: "dev.1"}
			}),
		},
		{
			name: "Success/EnglishAlone",

			request: valid(withEnglish(func(*core.PluginVersionPublishLanguage) {})),

			daoRequest: stored(func(request *dao.PluginVersionInsertRequest) {
				request.Languages = []dao.PluginVersionInsertLanguage{daoLanguage("en", english)}
			}),
		},
		{
			name: "Success/NoExport",

			request: valid(func(request *core.PluginVersionPublishRequest) { request.Exports = json.RawMessage(`{}`) }),

			daoRequest: stored(func(request *dao.PluginVersionInsertRequest) { request.Exports = json.RawMessage(`{}`) }),
		},
		{
			name: "Success/LargestModules",

			request: valid(func(request *core.PluginVersionPublishRequest) {
				request.UI = bytes.Repeat([]byte("a"), pluginUIBytesMax)
				request.Migration = bytes.Repeat([]byte("a"), pluginDocumentBytesMax)
			}),

			daoRequest: stored(func(request *dao.PluginVersionInsertRequest) {
				request.UI = bytes.Repeat([]byte("a"), pluginUIBytesMax)
				request.Migration = bytes.Repeat([]byte("a"), pluginDocumentBytesMax)
			}),
		},
		{
			name: "Success/LargestDocuments",

			request: valid(func(request *core.PluginVersionPublishRequest) {
				request.Schema = document(pluginDocumentBytesMax)
				request.Languages = map[string]core.PluginVersionPublishLanguage{"en": {
					Title:       strings.Repeat("a", pluginDocumentBytesMax),
					Description: strings.Repeat("a", pluginDocumentBytesMax),
					Messages:    document(pluginDocumentBytesMax),
				}}
			}),

			daoRequest: stored(func(request *dao.PluginVersionInsertRequest) {
				request.Schema = document(pluginDocumentBytesMax)
				request.Languages = []dao.PluginVersionInsertLanguage{{
					Language:    "en",
					Title:       strings.Repeat("a", pluginDocumentBytesMax),
					Description: strings.Repeat("a", pluginDocumentBytesMax),
					Messages:    document(pluginDocumentBytesMax),
				}}
			}),
		},
		{
			name: "Success/MostLanguages",

			request: valid(manyLanguages(100)),

			daoRequest: stored(func(request *dao.PluginVersionInsertRequest) {
				request.Languages = []dao.PluginVersionInsertLanguage{daoLanguage("en", english)}
				for i := range 99 {
					request.Languages = append(request.Languages, daoLanguage("x-a"+strconv.Itoa(i), english))
				}

				// The languages are stored in the order of their tags, which is not the order of their numbers.
				slices.SortFunc(request.Languages, func(a, b dao.PluginVersionInsertLanguage) int {
					return strings.Compare(a.Language, b.Language)
				})
			}),
		},
		{
			// A reference stays inside the schema, whatever it points at.
			name: "Success/Schema/LocalReferences",

			request: valid(withSchema(
				`{"$ref": "#/$defs/beat", "$defs": {"beat": {"$ref": "#beat"}, "named": {"$anchor": "beat"}}}`,
			)),

			daoRequest: stored(func(request *dao.PluginVersionInsertRequest) {
				request.Schema = json.RawMessage(
					`{"$ref": "#/$defs/beat", "$defs": {"beat": {"$ref": "#beat"}, "named": {"$anchor": "beat"}}}`,
				)
			}),
		},
		{
			// A property named after a keyword is no keyword, and neither is the content of a default, of an
			// enum or of a const.
			name: "Success/Schema/KeywordsAsData",

			request: valid(withSchema(`{
				"properties": {"format": {"type": "string"}, "$ref": {"type": "string"}, "$dynamicRef": true},
				"default": {"format": "date", "$ref": "https://example.com/schema.json"},
				"enum": [{"$dynamicAnchor": "node"}],
				"const": {"$dynamicRef": "#node"},
				"required": ["format"]
			}`)),

			daoRequest: stored(func(request *dao.PluginVersionInsertRequest) {
				request.Schema = json.RawMessage(`{
				"properties": {"format": {"type": "string"}, "$ref": {"type": "string"}, "$dynamicRef": true},
				"default": {"format": "date", "$ref": "https://example.com/schema.json"},
				"enum": [{"$dynamicAnchor": "node"}],
				"const": {"$dynamicRef": "#node"},
				"required": ["format"]
			}`)
			}),
		},
		{
			name: "Success/Schema/Deep",

			request: valid(withSchema(nested(5000, `{"type": "string"}`))),

			daoRequest: stored(func(request *dao.PluginVersionInsertRequest) {
				request.Schema = json.RawMessage(nested(5000, `{"type": "string"}`))
			}),
		},
		{
			name: "Error/Exists",

			request: valid(),

			daoRequest: stored(),
			daoErr:     dao.ErrPluginVersionExists,

			expectErr: core.ErrPluginVersionExists,
		},
		{
			name: "Error/Internal",

			request: valid(),

			daoRequest: stored(),
			daoErr:     errFoo,

			expectErr: errFoo,
		},

		refusedProof("OtherKey", &jsonKeysMock{keyID: "other-key", claims: proofClaims()}),
		refusedProof("OtherPlugin", &jsonKeysMock{keyID: apiKeyID, claims: proofClaims(withClaim("plugin", "timeline"))}),
		refusedProof("OtherVersion", &jsonKeysMock{keyID: apiKeyID, claims: proofClaims(withClaim("version", "1.10.1"))}),
		refusedProof("OtherUIDigest", &jsonKeysMock{
			keyID: apiKeyID, claims: proofClaims(withClaim("uiSHA256", digest(migrate))),
		}),
		// A digest is lowercase hex.
		refusedProof("UppercaseUIDigest", &jsonKeysMock{
			keyID: apiKeyID, claims: proofClaims(withClaim("uiSHA256", strings.ToUpper(digest(ui)))),
		}),
		refusedProof("OtherMigrationDigest", &jsonKeysMock{
			keyID: apiKeyID, claims: proofClaims(withClaim("migrationSHA256", digest(ui))),
		}),
		refusedProof("NoMigrationDigest", &jsonKeysMock{keyID: apiKeyID, claims: proofClaims(noMigrationClaim)}),
		refusedProof("MigrationDigestWithoutMigration", &jsonKeysMock{keyID: apiKeyID, claims: proofClaims()}, noMigration),
		refusedProof("EmptyMigrationDigestWithoutMigration", &jsonKeysMock{
			keyID: apiKeyID, claims: proofClaims(withClaim("migrationSHA256", "")),
		}, noMigration),
		// JSON Keys refuses the proof: another kind of token, an unknown or a leaked key, another purpose, a
		// signature that does not verify.
		refusedProof("Refused", &jsonKeysMock{
			err: fmt.Errorf("%w: invalid signature", servicejsonkeys.ErrTokenRejected),
		}),
		// A key publishes nothing it does not sign.
		refusedProof("Missing", nil, func(request *core.PluginVersionPublishRequest) { request.Proof = "" }),
		// Nothing vouches for the key of a proof sent without a request token.
		refusedProof("NoKey", nil, func(request *core.PluginVersionPublishRequest) { request.APIKeyID = "" }),
		{
			// JSON Keys failing is no fault of the publisher.
			name: "Error/Proof/JSONKeysFailure",

			request: valid(signed),

			jsonKeysMock: &jsonKeysMock{err: errFoo},

			expectErr:   errFoo,
			expectFault: true,
		},

		// The name and the version follow the rules of the registry's reads, which their own tests cover.
		invalid("Plugin", "Plugin", "dns_rfc1035_label", func(request *core.PluginVersionPublishRequest) {
			request.Plugin = "Scenes_Composer"
		}),
		invalid("Plugin/Missing", "Plugin", "required", func(request *core.PluginVersionPublishRequest) {
			request.Plugin = ""
		}),
		invalid("Version", "Version", "pluginversion", func(request *core.PluginVersionPublishRequest) {
			request.Version = "1.10.0+build.5"
		}),
		invalid("Version/Missing", "Version", "required", func(request *core.PluginVersionPublishRequest) {
			request.Version = ""
		}),

		invalid("Schema/Missing", "Schema", "required", func(request *core.PluginVersionPublishRequest) {
			request.Schema = nil
		}),
		invalid("Schema/TooLarge", "Schema", "max", func(request *core.PluginVersionPublishRequest) {
			request.Schema = document(pluginDocumentBytesMax + 1)
		}),
		invalid("Schema/Null", "Schema", "pluginschema", withSchema(`null`)),
		invalid("Schema/Boolean", "Schema", "pluginschema", withSchema(`true`)),
		invalid("Schema/Array", "Schema", "pluginschema", withSchema(`[{"type": "object"}]`)),
		invalid("Schema/RemoteReference", "Schema", "pluginschema", withSchema(
			`{"$ref": "https://example.com/schema.json"}`,
		)),
		invalid("Schema/RelativeReference", "Schema", "pluginschema", withSchema(`{"$ref": "beat.json#/$defs/beat"}`)),
		invalid("Schema/EmptyReference", "Schema", "pluginschema", withSchema(`{"$ref": ""}`)),
		invalid("Schema/ReferenceNotString", "Schema", "pluginschema", withSchema(`{"$ref": {"$defs": {}}}`)),
		invalid("Schema/DynamicReference", "Schema", "pluginschema", withSchema(`{"$dynamicRef": "#node"}`)),
		invalid("Schema/DynamicAnchor", "Schema", "pluginschema", withSchema(`{"$dynamicAnchor": "node"}`)),
		invalid("Schema/Format", "Schema", "pluginschema", withSchema(`{"type": "string", "format": "date"}`)),
		invalid("Schema/DeepFormat", "Schema", "pluginschema", withSchema(nested(5000, `{"format": "date"}`))),

		invalid("Exports/Missing", "Exports", "required", func(request *core.PluginVersionPublishRequest) {
			request.Exports = nil
		}),
		invalid("Exports/TooLarge", "Exports", "max", func(request *core.PluginVersionPublishRequest) {
			request.Exports = document(pluginDocumentBytesMax + 1)
		}),
		// The shared cases of the exports, below, cover each rule of an export.
		invalid("Exports/PathOutsideSubset", "Exports", "pluginexports", func(request *core.PluginVersionPublishRequest) {
			request.Exports = json.RawMessage(
				`{"scenes": {"name": "exports.scenes", "select": "$..scenes[*]", "id": "/id", "label": "/title"}}`,
			)
		}),
		invalid("Exports/Null", "Exports", "pluginexports", func(request *core.PluginVersionPublishRequest) {
			request.Exports = json.RawMessage(`null`)
		}),
		invalid("Exports/NotJSON", "Exports", "pluginexports", func(request *core.PluginVersionPublishRequest) {
			request.Exports = json.RawMessage(`{"scenes":`)
		}),

		invalid("UI/Missing", "UI", "required", func(request *core.PluginVersionPublishRequest) {
			request.UI = nil
		}),
		invalid("UI/Empty", "UI", "min", func(request *core.PluginVersionPublishRequest) {
			request.UI = []byte{}
		}),
		invalid("UI/TooLarge", "UI", "max", func(request *core.PluginVersionPublishRequest) {
			request.UI = bytes.Repeat([]byte("a"), pluginUIBytesMax+1)
		}),
		invalid("Migration/Empty", "Migration", "min", func(request *core.PluginVersionPublishRequest) {
			request.Migration = []byte{}
		}),
		invalid("Migration/TooLarge", "Migration", "max", func(request *core.PluginVersionPublishRequest) {
			request.Migration = bytes.Repeat([]byte("a"), pluginDocumentBytesMax+1)
		}),

		invalid("Proof/TooLong", "Proof", "max", func(request *core.PluginVersionPublishRequest) {
			signed(request)

			request.Proof = strings.Repeat("a", pluginProofMax+1)
		}),

		invalid("Languages/Missing", "Languages", "required", func(request *core.PluginVersionPublishRequest) {
			request.Languages = nil
		}),
		invalid("Languages/None", "Languages", "pluginenglish", func(request *core.PluginVersionPublishRequest) {
			request.Languages = map[string]core.PluginVersionPublishLanguage{}
		}),
		invalid("Languages/NoEnglish", "Languages", "pluginenglish", func(request *core.PluginVersionPublishRequest) {
			request.Languages = map[string]core.PluginVersionPublishLanguage{"fr": french}
		}),
		// The registry reads English as "en", in that spelling.
		invalid("Languages/EnglishOtherSpelling", "Languages", "pluginenglish", func(
			request *core.PluginVersionPublishRequest,
		) {
			request.Languages = map[string]core.PluginVersionPublishLanguage{"EN": english, "en-US": english}
		}),
		invalid("Languages/TooMany", "Languages", "max", manyLanguages(101)),
		invalid("Languages/NotATag", "Languages[french!]", "bcp47_language_tag", func(
			request *core.PluginVersionPublishRequest,
		) {
			request.Languages["french!"] = french
		}),
		invalid("Languages/OtherLanguageWithoutTitle", "Languages[fr].Title", "required", func(
			request *core.PluginVersionPublishRequest,
		) {
			untitled := french
			untitled.Title = ""
			request.Languages["fr"] = untitled
		}),

		invalid("English/NoTitle", "Languages[en].Title", "required", withEnglish(
			func(english *core.PluginVersionPublishLanguage) { english.Title = "" },
		)),
		invalid("English/BlankTitle", "Languages[en].Title", "notblank", withEnglish(
			func(english *core.PluginVersionPublishLanguage) { english.Title = " \t" },
		)),
		invalid("English/TitleTooLong", "Languages[en].Title", "max", withEnglish(
			func(english *core.PluginVersionPublishLanguage) {
				english.Title = strings.Repeat("a", pluginDocumentBytesMax+1)
			},
		)),
		invalid("English/NoDescription", "Languages[en].Description", "required", withEnglish(
			func(english *core.PluginVersionPublishLanguage) { english.Description = "" },
		)),
		invalid("English/BlankDescription", "Languages[en].Description", "notblank", withEnglish(
			func(english *core.PluginVersionPublishLanguage) { english.Description = "\n" },
		)),
		invalid("English/DescriptionTooLong", "Languages[en].Description", "max", withEnglish(
			func(english *core.PluginVersionPublishLanguage) {
				english.Description = strings.Repeat("a", pluginDocumentBytesMax+1)
			},
		)),
		invalid("English/NoMessages", "Languages[en].Messages", "required", withEnglish(
			func(english *core.PluginVersionPublishLanguage) { english.Messages = nil },
		)),
		invalid("English/MessagesTooLarge", "Languages[en].Messages", "max", withEnglish(
			func(english *core.PluginVersionPublishLanguage) {
				english.Messages = document(pluginDocumentBytesMax + 1)
			},
		)),
		invalid("English/MessagesNull", "Languages[en].Messages", "jsonobject", withEnglish(
			func(english *core.PluginVersionPublishLanguage) { english.Messages = json.RawMessage(`null`) },
		)),
		invalid("English/MessagesArray", "Languages[en].Messages", "jsonobject", withEnglish(
			func(english *core.PluginVersionPublishLanguage) { english.Messages = json.RawMessage(`["Scenes"]`) },
		)),
		invalid("English/TitleNul", "Languages[en].Title", "nonul", withEnglish(
			func(english *core.PluginVersionPublishLanguage) { english.Title = "Scenes\x00Composer" },
		)),
		invalid("English/DescriptionNul", "Languages[en].Description", "nonul", withEnglish(
			func(english *core.PluginVersionPublishLanguage) { english.Description = "Compose\x00scenes" },
		)),

		// InstanceContentUpdate covers each rule of the document check. These prove that every document of a
		// version takes it, under its path in the body.
		unstorable("Schema/Unstorable", "schema", "nonul", withSchema(`{"title": "a\u0000b"}`)),
		unstorable("Exports/Unstorable", "exports", "nolonesurrogate", func(
			request *core.PluginVersionPublishRequest,
		) {
			request.Exports = json.RawMessage(
				`{"scenes": {"name": "\ud83d", "select": "$", "id": "/id", "label": "/title"}}`,
			)
		}),
		// Seventeen numbers of 16001 digits once the database writes them in full, in a document of 150 bytes.
		unstorable("Languages/OtherLanguageMessagesStoredTooLarge", "languages[fr].messages", "maxbytes", func(
			request *core.PluginVersionPublishRequest,
		) {
			grown := french
			grown.Messages = json.RawMessage(`{"sizes": [` + strings.Repeat("1e16000, ", 16) + `1e16000]}`)
			request.Languages["fr"] = grown
		}),
	}

	// A keyword the dialect lacks is refused under every keyword that holds a subschema.
	for keyword, schema := range pluginSchemaOf(`{"format": "date"}`) {
		testCases = append(testCases, invalid("Schema/FormatUnder/"+keyword, "Schema", "pluginschema", withSchema(schema)))
	}

	for keyword, schema := range pluginSchemaOf(`{"$ref": "https://example.com/schema.json"}`) {
		testCases = append(
			testCases, invalid("Schema/RemoteReferenceUnder/"+keyword, "Schema", "pluginschema", withSchema(schema)),
		)
	}

	// The shared cases of the exports run under the names their file gives them.
	var exportsCases []struct {
		Name     string          `json:"name"`
		Valid    bool            `json:"valid"`
		Document json.RawMessage `json:"document"`
	}

	require.NoError(t, json.Unmarshal(pluginExportsCases, &exportsCases))
	require.NotEmpty(t, exportsCases)

	for _, exportsCase := range exportsCases {
		shared := testCase{
			name: "Exports/" + exportsCase.Name,
			request: valid(func(request *core.PluginVersionPublishRequest) {
				request.Exports = exportsCase.Document
			}),
		}

		if exportsCase.Valid {
			shared.daoRequest = stored(func(request *dao.PluginVersionInsertRequest) {
				request.Exports = exportsCase.Document
			})
		} else {
			shared.expectErr, shared.expectField, shared.expectRule = core.ErrInvalidRequest, "Exports", "pluginexports"
		}

		testCases = append(testCases, shared)
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockJSONKeys := coremocks.NewMockPluginVersionPublishServiceJSONKeys(t)

			if testCase.jsonKeysMock != nil {
				mockJSONKeys.EXPECT().
					VerifyProof(mock.Anything, mock.MatchedBy(func(request *servicejsonkeys.VerifyProofRequest) bool {
						return request.Proof == testCase.request.Proof
					})).
					RunAndReturn(func(
						_ context.Context, request *servicejsonkeys.VerifyProofRequest,
					) (*servicejsonkeys.APIKeyProof, error) {
						if testCase.jsonKeysMock.err != nil {
							return nil, testCase.jsonKeysMock.err
						}

						// JSON Keys decodes the claims of a proof into the value the service hands it.
						encoded, err := json.Marshal(testCase.jsonKeysMock.claims)
						if err != nil {
							panic(err)
						}

						err = json.Unmarshal(encoded, request.Claims)
						if err != nil {
							panic(err)
						}

						return &servicejsonkeys.APIKeyProof{KeyID: testCase.jsonKeysMock.keyID}, nil
					})
			}

			mockDao := coremocks.NewMockPluginVersionPublishDao(t)

			if testCase.daoRequest != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.PluginVersionInsertRequest) bool {
						expect := *testCase.daoRequest
						expect.Now = request.Now

						if !recent(request.Now) {
							t.Error("the service captures the instant of the publication")
						}

						return reflect.DeepEqual(&expect, request)
					})).
					Return(testCase.daoErr)
			}

			service := core.NewPluginVersionPublish(mockDao, mockJSONKeys)

			err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectFault {
				require.NotErrorIs(t, err, core.ErrInvalidRequest)
			}

			if testCase.expectDocument != nil {
				refused, ok := errors.AsType[*core.FieldError](err)
				require.True(t, ok)
				require.Equal(t, testCase.expectDocument, refused)
			}

			if testCase.expectField == "" {
				return
			}

			fields, ok := errors.AsType[validator.ValidationErrors](err)
			require.True(t, ok)
			require.Len(t, fields, 1)
			require.Equal(t, "PluginVersionPublishRequest."+testCase.expectField, fields[0].Namespace())
			require.Equal(t, testCase.expectRule, fields[0].Tag())
		})
	}

	// The cases above stand in for JSON Keys. These run its own verifier on a proof a key signed, so the
	// service reads the claims as a publisher writes them.
	const purpose = servicejsonkeys.APIKeyPurposePublishNarrativeEnginePlugin

	publisher := servicejsonkeystest.NewAPIKey(t, purpose)
	unknown := servicejsonkeystest.NewAPIKey(t, purpose)

	verifier, err := servicejsonkeys.NewAPIKeyVerifier(servicejsonkeystest.APIKeys{publisher}, purpose)
	require.NoError(t, err)

	signedCases := []struct {
		name string

		// key signs the proof of a request the publisher's key authenticated.
		key *servicejsonkeystest.APIKey

		expectErr error
	}{
		{name: "Success/Proof/Signed", key: publisher},
		// JSON Keys serves no such key, and refuses what it signed.
		{name: "Error/Proof/SignedByUnknownKey", key: unknown, expectErr: core.ErrPluginProof},
	}

	for _, testCase := range signedCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			token := testCase.key.Sign(t, servicejsonkeystest.TypProof, testCase.key.ProofClaims(proofClaims()))

			mockDao := coremocks.NewMockPluginVersionPublishDao(t)

			if testCase.expectErr == nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.PluginVersionInsertRequest) bool {
						return lo.FromPtr(request.Proof) == token && lo.FromPtr(request.APIKeyID) == publisher.ID
					})).
					Return(nil)
			}

			request := valid(func(request *core.PluginVersionPublishRequest) {
				request.Proof, request.APIKeyID = token, publisher.ID
			})

			err := core.NewPluginVersionPublish(mockDao, verifier).Exec(t.Context(), request)
			require.ErrorIs(t, err, testCase.expectErr)
		})
	}
}
