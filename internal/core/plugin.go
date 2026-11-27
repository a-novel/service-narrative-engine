package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// PluginModuleKind names a JavaScript module a version of a plugin publishes.
type PluginModuleKind string

const (
	// PluginModuleUI is the module that renders a plugin.
	PluginModuleUI PluginModuleKind = "ui"
	// PluginModuleMigration is the module that brings content from the previous version of a plugin.
	PluginModuleMigration PluginModuleKind = "migration"
)

// PluginLanguageEnglish is the language every version of a plugin is translated in, as a BCP 47 tag. A read
// falls back on it for a plugin that lacks the language asked for.
const PluginLanguageEnglish = "en"

// Plugin is a plugin of the registry, as its latest release describes it.
type Plugin struct {
	// Name identifies the plugin across its versions. It is a DNS label (RFC 1035): lowercase letters, digits
	// and hyphens, from a letter to a letter or a digit, 63 characters at most.
	Name string
	// Version is the plugin's latest release, as a semantic version. A prerelease is never one.
	Version string

	// Title names the plugin, in the language asked for, or in English when the release lacks it.
	Title string
	// Description says what the plugin does, in the language of Title.
	Description string
}

// PluginPage is a page of the registry's plugins.
type PluginPage struct {
	Plugins []*Plugin

	// Total counts every plugin that has a release, so a client paging through them knows when it holds them
	// all. A page past the end reports 0.
	Total int
}

// PluginManifest describes one version of a plugin: everything it publishes but its modules.
type PluginManifest struct {
	// Plugin is the plugin's name.
	Plugin string
	// Version is the version described, as a semantic version.
	Version string

	// Schema is the JSON Schema the content of an instance follows.
	Schema json.RawMessage
	// Exports holds the lists of linkable items an instance exposes, by name.
	Exports json.RawMessage
	// HasMigration reports whether the version ships a migration module.
	HasMigration bool

	// SchemaChecksum is the SHA-256 of the stored schema in lowercase hex. Two versions with the same one share
	// their schema.
	SchemaChecksum string
	// UIChecksum is the SHA-256 of the UI module in lowercase hex.
	UIChecksum string

	// Proof is the token the publisher's API key signed for the version: a compact JWT of type "proof+jwt",
	// whose claims name the plugin and the version, and carry the SHA-256 of each module. Nil for a version
	// published without a key.
	Proof *string
	// APIKeyID is the ID of the API key that signed Proof, which JSON Keys serves by that ID. Nil when Proof
	// is.
	APIKeyID *string

	PublishedAt time.Time
}

// PluginModule is one JavaScript module of a version of a plugin.
type PluginModule struct {
	// Code holds the module's source.
	Code []byte
	// Checksum is the SHA-256 of Code in lowercase hex.
	Checksum string

	// PublishedAt is when the module's version was published. A module never changes after it.
	PublishedAt time.Time
}

// PluginMessages holds the translations of one version of a plugin in one language.
type PluginMessages struct {
	// Language is the language of the translations, as a BCP 47 tag.
	Language string

	// Title names the plugin.
	Title string
	// Description says what the plugin does.
	Description string
	// Messages holds the translations of the plugin's interface and of its schema, by key.
	Messages json.RawMessage
}

// PluginVersion is one release of a plugin, as an upgrade steps through it.
type PluginVersion struct {
	// Version is the release, as a semantic version.
	Version string

	// HasMigration reports whether the release ships a migration module, which an upgrade runs to reach it.
	HasMigration bool
	// SchemaChecksum is the SHA-256 of the stored schema in lowercase hex. Two releases with the same one
	// share their schema.
	SchemaChecksum string
}

// PluginVersionPage is a page of the releases of a plugin.
type PluginVersionPage struct {
	Versions []*PluginVersion

	// Total counts every release the request matches, so a client paging through them knows when it holds
	// them all. A page past the end reports 0.
	Total int
}

// pluginVersionPattern matches a semantic version (Semantic Versioning 2.0.0, semver.org) without build
// metadata, which the registry does not store: three numbers, then an optional hyphen and dot-separated
// prerelease identifiers. A number takes no leading zero.
var pluginVersionPattern = regexp.MustCompile(
	`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
		`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?$`,
)

// parsePluginVersion splits a version into the columns the registry keys it by, and leaves the plugin unset. It
// refuses anything but a semantic version without build metadata whose numbers fit those columns, with an
// [ErrInvalidRequest] that says which of the two the version breaks.
func parsePluginVersion(version string) (dao.PluginVersionKey, error) {
	parts := pluginVersionPattern.FindStringSubmatch(version)
	if parts == nil {
		return dao.PluginVersionKey{}, fmt.Errorf(
			"%w: not a semantic version without build metadata: %q", ErrInvalidRequest, version,
		)
	}

	var numbers [3]int

	for i := range numbers {
		// The registry stores each number in a 32-bit column.
		number, err := strconv.ParseInt(parts[i+1], 10, 32)
		if err != nil {
			return dao.PluginVersionKey{}, fmt.Errorf(
				"%w: parse a number of version %q: %w", ErrInvalidRequest, version, err,
			)
		}

		numbers[i] = int(number)
	}

	return dao.PluginVersionKey{Major: numbers[0], Minor: numbers[1], Patch: numbers[2], Prerelease: parts[4]}, nil
}

// formatPluginVersion spells the version a key names as a semantic version, such as 1.2.0-dev.1.
func formatPluginVersion(key dao.PluginVersionKey) string {
	version := strconv.Itoa(key.Major) + "." + strconv.Itoa(key.Minor) + "." + strconv.Itoa(key.Patch)
	if key.Prerelease != "" {
		version += "-" + key.Prerelease
	}

	return version
}
