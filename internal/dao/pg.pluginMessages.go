package dao

import (
	"encoding/json"

	"github.com/uptrace/bun"
)

// PluginMessages holds the translations of one version of a plugin in one language.
type PluginMessages struct {
	bun.BaseModel `bun:"table:plugin_messages,alias:plugin_messages"`

	PluginVersionKey

	// Language is the language of the translations, as a BCP 47 tag.
	Language string `bun:"language,pk"`

	// Title names the plugin in the list of plugins.
	Title string `bun:"title"`
	// Description says what the plugin does, in the list of plugins.
	Description string `bun:"description"`
	// Messages holds the translations of the plugin's interface and of its schema, by key.
	Messages json.RawMessage `bun:"messages,type:jsonb"`
}
