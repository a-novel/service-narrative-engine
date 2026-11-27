package dao

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// InstanceDescriptor is what a project knows of an instance without reading its content.
type InstanceDescriptor struct {
	ID        uuid.UUID `bun:"id,pk,type:uuid"`
	ProjectID uuid.UUID `bun:"project_id,type:uuid"`

	// Plugin is the name of the plugin, as the registry holds it.
	Plugin string `bun:"plugin"`
	// Version is the exact version of the plugin the content follows, as a semantic version.
	Version string `bun:"version"`
	// Name is the display name the writer gave the instance.
	Name string `bun:"name"`

	CreatedAt time.Time `bun:"created_at"`
	// UpdatedAt is when the version or the content last changed. A rename leaves it as it was.
	UpdatedAt time.Time `bun:"updated_at"`
}

// Instance is one use of a plugin in a project, with its content.
type Instance struct {
	bun.BaseModel `bun:"table:instances,alias:instances"`

	InstanceDescriptor

	// Content is the document the plugin edits.
	Content json.RawMessage `bun:"content,type:jsonb"`
}
