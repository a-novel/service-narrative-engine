package handlers

import (
	"encoding/json"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Plugin is the REST representation of a plugin of the registry, as its latest release describes it.
type Plugin struct {
	// Name identifies the plugin across its versions.
	Name string `json:"name"`
	// Version is the plugin's latest release, as a semantic version.
	Version string `json:"version"`

	// Title names the plugin, in the language asked for, or in English when the release lacks it.
	Title string `json:"title"`
	// Description says what the plugin does, in the language of Title.
	Description string `json:"description"`
}

// PluginPage is the REST representation of a page of the registry's plugins.
type PluginPage struct {
	Plugins []Plugin `json:"plugins"`

	// Total counts every plugin that has a release, so a client paging through them knows when it holds them
	// all.
	Total int `json:"total"`
}

func loadPluginPage(page *core.PluginPage) PluginPage {
	return PluginPage{
		Plugins: lo.Map(page.Plugins, func(plugin *core.Plugin, _ int) Plugin {
			return Plugin{
				Name:        plugin.Name,
				Version:     plugin.Version,
				Title:       plugin.Title,
				Description: plugin.Description,
			}
		}),
		Total: page.Total,
	}
}

// PluginManifest is the REST representation of one version of a plugin: everything it publishes but its
// modules.
type PluginManifest struct {
	// Plugin is the plugin's name.
	Plugin string `json:"plugin"`
	// Version is the version described, as a semantic version.
	Version string `json:"version"`

	// Schema is the JSON Schema the content of an instance follows.
	Schema json.RawMessage `json:"schema"`
	// Exports holds the lists of linkable items an instance exposes, by name.
	Exports json.RawMessage `json:"exports"`
	// HasMigration reports whether the version ships a migration module.
	HasMigration bool `json:"hasMigration"`

	// SchemaChecksum is the SHA-256 of the stored schema in lowercase hex.
	SchemaChecksum string `json:"schemaChecksum"`
	// UIChecksum is the SHA-256 of the UI module in lowercase hex.
	UIChecksum string `json:"uiChecksum"`

	PublishedAt time.Time `json:"publishedAt"`
	// PublishedBy is the identity that published the version.
	PublishedBy string `json:"publishedBy"`
}

func loadPluginManifest(manifest *core.PluginManifest) PluginManifest {
	return PluginManifest{
		Plugin:         manifest.Plugin,
		Version:        manifest.Version,
		Schema:         manifest.Schema,
		Exports:        manifest.Exports,
		HasMigration:   manifest.HasMigration,
		SchemaChecksum: manifest.SchemaChecksum,
		UIChecksum:     manifest.UIChecksum,
		PublishedAt:    manifest.PublishedAt,
		PublishedBy:    manifest.PublishedBy,
	}
}

// PluginMessages is the REST representation of the translations of one version of a plugin in one language.
type PluginMessages struct {
	// Language is the language of the translations, as a BCP 47 tag.
	Language string `json:"language"`

	// Title names the plugin.
	Title string `json:"title"`
	// Description says what the plugin does.
	Description string `json:"description"`
	// Messages holds the translations of the plugin's interface and of its schema, by key.
	Messages json.RawMessage `json:"messages"`
}

// PluginVersion is the REST representation of one release of a plugin, as an upgrade steps through it.
type PluginVersion struct {
	// Version is the release, as a semantic version.
	Version string `json:"version"`

	// HasMigration reports whether the release ships a migration module, which an upgrade runs to reach it.
	HasMigration bool `json:"hasMigration"`
	// SchemaChecksum is the SHA-256 of the stored schema in lowercase hex. Two releases with the same one
	// share their schema.
	SchemaChecksum string `json:"schemaChecksum"`
}

// PluginVersionPage is the REST representation of a page of the releases of a plugin.
type PluginVersionPage struct {
	Versions []PluginVersion `json:"versions"`

	// Total counts every release the request matches, so a client paging through them knows when it holds
	// them all.
	Total int `json:"total"`
}

func loadPluginVersionPage(page *core.PluginVersionPage) PluginVersionPage {
	return PluginVersionPage{
		Versions: lo.Map(page.Versions, func(version *core.PluginVersion, _ int) PluginVersion {
			return PluginVersion{
				Version:        version.Version,
				HasMigration:   version.HasMigration,
				SchemaChecksum: version.SchemaChecksum,
			}
		}),
		Total: page.Total,
	}
}
