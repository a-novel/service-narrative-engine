package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// SchemaMeta contains metadata about a schema version without the actual data content.
type SchemaMeta struct {
	bun.BaseModel `bun:"table:schemas"`

	// ID of this schema version.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// ProjectID links multiple versions of the same schema together.
	ProjectID uuid.UUID `bun:"project_id,type:uuid"`
	// Owner is the ID of the user who owns this schema version (nullable).
	Owner *uuid.UUID `bun:"owner,type:uuid"`

	// ModuleID is the ID of the module used to create the schema.
	ModuleID string `bun:"module_id"`
	// ModuleNamespace is the namespace of the module used to create the schema.
	ModuleNamespace string `bun:"module_namespace"`
	// ModuleVersion is the version of the module used to create the schema.
	ModuleVersion string `bun:"module_version"`
	// ModulePreversion is the preversion of the module used to create the schema.
	ModulePreversion string `bun:"module_preversion"`

	// Source is the source of this schema version.
	Source SchemaSource `bun:"source,type:schema_source"`

	// IsLatest indicates if this is the most recent version of the schema for the project/module combination.
	IsLatest bool `bun:"is_latest"`
	// IsNil indicates if the data is nil (NULL in the database). Empty objects do not count as nil.
	IsNil bool `bun:"is_nil"`

	CreatedAt time.Time `bun:"created_at"`
}
