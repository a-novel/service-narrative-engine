package dao

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// InstanceLinks is the static links an instance stores: the ones whose source is one of its items.
type InstanceLinks struct {
	// InstanceID is the instance that stores the links, their importer.
	InstanceID uuid.UUID `bun:"id,type:uuid"`

	// Links is a JSON array of links, each a source, a destination and their relationship. It is empty for an
	// instance that never wrote its links.
	Links json.RawMessage `bun:"links,type:jsonb"`

	// UpdatedAt is when the links last changed. It is nil for an instance that never wrote its links.
	UpdatedAt *time.Time `bun:"links_updated_at"`
}
