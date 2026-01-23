package dao

import (
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/uptrace/bun"

	"github.com/a-novel/service-narrative-engine/internal/models"
)

// Module represents a single unit of a story Engine workflow. It contains data used to shape the final story.
type Module struct {
	bun.BaseModel `bun:"table:modules"`

	// ID of the module, as an uri-safe string.
	ID string `bun:"id,pk"`
	// Namespace to which the module belongs.
	Namespace string `bun:"namespace,pk"`
	// Version of the module.
	Version string `bun:"version,pk"`
	// Preversion of the module (e.g., "-beta-1", "-rc-1"). Empty string for stable versions.
	Preversion string `bun:"preversion,pk"`

	// Description of the module.
	Description string `bun:"description"`

	// Schema defines the shape of the module output. It must be compatible with openAI Api structured outputs:
	// https://platform.openai.com/docs/guides/structured-outputs
	Schema jsonschema.Schema `bun:"schema,type:json"`
	// UI definition to interact with the module.
	UI models.ModuleUi `bun:"ui,type:json"`

	CreatedAt time.Time `bun:"created_at"`
}
