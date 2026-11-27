package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// ErrPluginProof is returned by [PluginVersionPublish] when the proof of a version does not vouch for it, and
// when a version carries a proof without an API key, or a key without a proof. It matches [ErrInvalidRequest].
var ErrPluginProof = fmt.Errorf("%w: plugin proof refused", ErrInvalidRequest)

// PluginVersionPublishDao stores one version of a plugin with its translations.
type PluginVersionPublishDao interface {
	Exec(ctx context.Context, request *dao.PluginVersionInsertRequest) error
}

// PluginVersionPublishServiceJSONKeys verifies a proof through JSON Keys, which serves the API key that signed
// it, and decodes its claims. A proof it refuses fails with an error matching
// [servicejsonkeys.ErrTokenRejected]. servicejsonkeys.APIKeyVerifier satisfies it, so the method keeps that
// published client's name instead of Exec.
//
// nosemgrep: a-novel-dep-interface-method-must-be-exec
type PluginVersionPublishServiceJSONKeys interface {
	VerifyProof(
		ctx context.Context, req *servicejsonkeys.VerifyProofRequest,
	) (*servicejsonkeys.APIKeyProof, error)
}

// pluginDocumentBytes caps each JSON document of a version as the database stores it, with its numbers written
// in full: its schema, its exports and the messages of a language. The tags of [PluginVersionPublishRequest]
// hold the document to the same cap as sent.
const pluginDocumentBytes = 262144

// PluginVersionPublishLanguage holds the translations of a [PluginVersionPublishRequest] in one language.
type PluginVersionPublishLanguage struct {
	// Title names the plugin in the list of plugins, in 262144 characters at most, none of them NUL.
	Title string `validate:"required,notblank,max=262144,nonul"`
	// Description says what the plugin does, in 262144 characters at most, none of them NUL.
	Description string `validate:"required,notblank,max=262144,nonul"`
	// Messages holds the translations of the plugin's interface and of its schema, by key: a JSON object of
	// 256 KiB at most.
	Messages json.RawMessage `validate:"required,max=262144,jsonobject"`
}

// PluginVersionPublishRequest holds one whole version of a plugin.
type PluginVersionPublishRequest struct {
	// Plugin is the plugin's name, a DNS label (RFC 1035).
	Plugin string `validate:"required,dns_rfc1035_label"`
	// Version is the version to publish, as a semantic version without build metadata. A prerelease, such as
	// 1.2.0-dev.1, publishes as a release does.
	Version string `validate:"required,pluginversion"`

	// Schema is the JSON Schema the content of an instance follows, of 256 KiB at most. Its subschemas only
	// reference the schema itself, and use none of "$dynamicRef", "$dynamicAnchor" and "format".
	Schema json.RawMessage `validate:"required,max=262144,pluginschema"`
	// Exports holds the lists of linkable items an instance exposes, by name, in 256 KiB at most.
	Exports json.RawMessage `validate:"required,max=262144,pluginexports"`
	// UI is the JavaScript module that renders the plugin, of 1 MiB at most.
	UI []byte `validate:"required,min=1,max=1048576"`
	// Migration is the JavaScript module that brings content from the previous version to this one, of 256 KiB
	// at most. Nil when the version needs none.
	Migration []byte `validate:"omitempty,min=1,max=262144"`

	// Languages holds the translations of the version by language, a BCP 47 tag: [PluginLanguageEnglish], and
	// 99 other languages at most.
	//
	//nolint:lll // A struct tag holds on one line.
	Languages map[string]PluginVersionPublishLanguage `validate:"required,pluginenglish,max=100,dive,keys,bcp47_language_tag,endkeys"`

	// Proof is the token the API key of the request signed for the version: a compact JWT of type "proof+jwt",
	// of 4096 characters at most. Its claims name the plugin and the version, and carry the SHA-256 of each
	// module in lowercase hex. Empty when no key authenticated the request.
	Proof string `validate:"omitempty,max=4096"`
	// APIKeyID is the ID of the API key that authenticated the request. Empty for a request the route took
	// without a key, as a local stack does. The route sets it, never the caller.
	APIKeyID string
}

// pluginProofClaims are the claims the proof of a version adds to the registered ones, which JSON Keys checks.
type pluginProofClaims struct {
	// Plugin and Version name the version the proof vouches for.
	Plugin  string `json:"plugin"`
	Version string `json:"version"`

	// UISHA256 is the SHA-256 of the UI module, in lowercase hex.
	UISHA256 string `json:"uiSHA256"` //nolint:tagliatelle // The proof names the claim.
	// MigrationSHA256 is the SHA-256 of the migration module, in lowercase hex. The proof of a version
	// without a migration leaves it out.
	MigrationSHA256 *string `json:"migrationSHA256"` //nolint:tagliatelle // The proof names the claim.
}

// PluginVersionPublish adds one version of a plugin to the registry, whole: its manifest, its modules and its
// translations, or nothing. A version never changes once published, so publishing one the registry holds
// returns [ErrPluginVersionExists] and leaves it as it is.
//
// The route verifies its caller, an API key. The version carries a proof that key signed, which the registry
// stores with the ID of the key once it vouches for the version, and returns [ErrPluginProof] otherwise. A
// version published without a key stores neither.
type PluginVersionPublish struct {
	dao      PluginVersionPublishDao
	jsonKeys PluginVersionPublishServiceJSONKeys
}

func NewPluginVersionPublish(
	dao PluginVersionPublishDao, jsonKeys PluginVersionPublishServiceJSONKeys,
) *PluginVersionPublish {
	return &PluginVersionPublish{dao: dao, jsonKeys: jsonKeys}
}

func (service *PluginVersionPublish) Exec(ctx context.Context, request *PluginVersionPublishRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "core.PluginVersionPublish")
	defer span.End()

	span.SetAttributes(
		attribute.String("plugin.name", request.Plugin),
		attribute.String("plugin.version", request.Version),
	)

	err := validate.Struct(request)
	if err != nil {
		return otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	err = checkPluginDocuments(request)
	if err != nil {
		return otel.ReportError(span, err)
	}

	key, err := parsePluginVersion(request.Version)
	if err != nil {
		return otel.ReportError(span, err)
	}

	key.Plugin = request.Plugin

	err = service.checkProof(ctx, request)
	if err != nil {
		return otel.ReportError(span, err)
	}

	// Go randomizes the order of a map, and the statement stores the languages in the order it receives them.
	languages := make([]dao.PluginVersionInsertLanguage, 0, len(request.Languages))

	for _, language := range slices.Sorted(maps.Keys(request.Languages)) {
		translations := request.Languages[language]

		languages = append(languages, dao.PluginVersionInsertLanguage{
			Language:    language,
			Title:       translations.Title,
			Description: translations.Description,
			Messages:    translations.Messages,
		})
	}

	err = service.dao.Exec(ctx, &dao.PluginVersionInsertRequest{
		PluginVersionKey: key,
		Schema:           request.Schema,
		Exports:          request.Exports,
		Migration:        request.Migration,
		UI:               request.UI,
		Proof:            lo.EmptyableToPtr(request.Proof),
		APIKeyID:         lo.EmptyableToPtr(request.APIKeyID),
		Languages:        languages,
		// The registry keeps whole seconds, and PostgreSQL rounds what it is given. A whole second already
		// past keeps the publication, and the Last-Modified of its modules, from standing ahead of the
		// clock (RFC 9110, section 8.8.2.1).
		Now: time.Now().Truncate(time.Second),
	})
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("insert version: %w", err))
	}

	return nil
}

// checkPluginDocuments returns a [FieldError] unless the database stores each JSON document of a version as it
// is, within [pluginDocumentBytes]: its schema, its exports and the messages of each language. A version caps
// no depth of its own: its tags already refused what the standard library does not read.
func checkPluginDocuments(request *PluginVersionPublishRequest) error {
	documents := map[string]json.RawMessage{"schema": request.Schema, "exports": request.Exports}

	for language, translations := range request.Languages {
		documents["languages["+language+"].messages"] = translations.Messages
	}

	// Go randomizes the order of a map, and the first refused document is the one reported.
	for _, path := range slices.Sorted(maps.Keys(documents)) {
		err := checkDocument(path, documents[path], pluginDocumentBytes, math.MaxInt)
		if err != nil {
			return err
		}
	}

	return nil
}

// checkProof returns [ErrPluginProof] unless the proof of a version vouches for it: the API key of the
// request signed it, for this plugin, this version and the modules the registry stores. A request no key
// authenticated carries no proof, and passes.
func (service *PluginVersionPublish) checkProof(ctx context.Context, request *PluginVersionPublishRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "core.PluginVersionPublish(checkProof)")
	defer span.End()

	if request.APIKeyID == "" && request.Proof == "" {
		return nil
	}

	// A proof no key vouches for is as good as none, and a key publishes nothing it does not sign.
	if request.APIKeyID == "" || request.Proof == "" {
		return otel.ReportError(span, fmt.Errorf(
			"%w: a proof and the api key of the request come together", ErrPluginProof,
		))
	}

	var claims pluginProofClaims

	proof, err := service.jsonKeys.VerifyProof(ctx, &servicejsonkeys.VerifyProofRequest{
		Proof: request.Proof, Claims: &claims,
	})
	if errors.Is(err, servicejsonkeys.ErrTokenRejected) {
		err = errors.Join(err, ErrPluginProof)
	}

	if err != nil {
		return otel.ReportError(span, fmt.Errorf("verify proof: %w", err))
	}

	mismatch := pluginProofMismatch(proof.KeyID, &claims, request)
	if mismatch != "" {
		return otel.ReportError(span, fmt.Errorf("%w: %s", ErrPluginProof, mismatch))
	}

	return nil
}

// pluginProofMismatch says how a proof JSON Keys verified differs from the version a request publishes: the
// key that signed it, then its claims, which name each module by the SHA-256 of the bytes the registry
// stores. It returns an empty string when the proof vouches for the version.
func pluginProofMismatch(keyID string, claims *pluginProofClaims, request *PluginVersionPublishRequest) string {
	switch {
	case keyID != request.APIKeyID:
		return "another api key than the request's signed it"
	case claims.Plugin != request.Plugin:
		return "it names another plugin"
	case claims.Version != request.Version:
		return "it names another version"
	case claims.UISHA256 != pluginModuleSHA256(request.UI):
		return "it carries another digest of the ui module"
	case request.Migration == nil && claims.MigrationSHA256 != nil:
		return "it carries the digest of a migration module the version lacks"
	case request.Migration != nil && lo.FromPtr(claims.MigrationSHA256) != pluginModuleSHA256(request.Migration):
		return "it carries another digest of the migration module, or none"
	default:
		return ""
	}
}

// pluginModuleSHA256 returns the SHA-256 of a module in lowercase hex, as the registry serves its checksum.
func pluginModuleSHA256(module []byte) string {
	sum := sha256.Sum256(module)

	return hex.EncodeToString(sum[:])
}
