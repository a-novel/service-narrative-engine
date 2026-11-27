package dao

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ProjectLinks is the links of a project, stored whole. A project that never wrote its links holds none.
type ProjectLinks struct {
	bun.BaseModel `bun:"table:project_links,alias:project_links"`

	ProjectID uuid.UUID `bun:"project_id,pk,type:uuid"`

	// Links is a JSON array of links, each a source, a destination and their relationship.
	Links json.RawMessage `bun:"links,type:jsonb"`

	// UpdatedAt is when the links last changed.
	UpdatedAt time.Time `bun:"updated_at"`
}
