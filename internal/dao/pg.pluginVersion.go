package dao

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

// PluginVersionKey names one version of a plugin, as the registry's tables key it.
type PluginVersionKey struct {
	// Plugin is the plugin's name, which identifies it across its versions.
	Plugin string `bun:"plugin,pk"`

	Major int `bun:"major,pk"`
	Minor int `bun:"minor,pk"`
	Patch int `bun:"patch,pk"`
	// Prerelease holds the prerelease identifiers of a semantic version, such as dev.1. Empty for a release.
	Prerelease string `bun:"prerelease,pk"`
}

// PluginVersion is one published version of a plugin. The registry never updates or deletes one.
type PluginVersion struct {
	bun.BaseModel `bun:"table:plugin_versions,alias:plugin_versions"`

	PluginVersionKey

	// Schema is the JSON Schema the content of an instance follows.
	Schema json.RawMessage `bun:"schema,type:jsonb"`
	// Exports holds the lists of linkable items an instance exposes, by name.
	Exports json.RawMessage `bun:"exports,type:jsonb"`
	// Migration is the JavaScript module that brings content from the previous version to this one. Nil when
	// the version needs none.
	Migration []byte `bun:"migration"`
	// UI is the JavaScript module that renders the plugin.
	UI []byte `bun:"ui"`

	PublishedAt time.Time `bun:"published_at"`
	// PublishedBy is the identity that published the version.
	PublishedBy string `bun:"published_by"`
}
