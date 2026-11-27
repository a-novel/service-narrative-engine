package handlers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
)

// Identifier is the REST representation of one end of a link: an item of a project.
type Identifier struct {
	// ID is the instance that holds the item.
	ID uuid.UUID `json:"id"`
	// Target is the address of the item inside that instance.
	Target string `json:"target"`
}

// Link is the REST representation of a link between two items of one project.
type Link struct {
	Source      Identifier `json:"source"`
	Destination Identifier `json:"destination"`
	// Relationship says what the source is to the destination, in the words of the plugin that wrote the link.
	Relationship string `json:"relationship"`
}

// ProjectLinks is the REST representation of the links of a project.
type ProjectLinks struct {
	// Links is the list of [Link] as the project stores it, which a read returns without decoding it.
	Links json.RawMessage `json:"links"`

	// UpdatedAt is when the links last changed, null for a project that never wrote its links. Its JSON keeps
	// the microseconds the service stores, and a write sends it back as it read it.
	UpdatedAt *time.Time `json:"updatedAt"`
}

// WrittenProjectLinks is the REST representation of a write of the links, which the next write names.
type WrittenProjectLinks struct {
	UpdatedAt time.Time `json:"updatedAt"`
}

func loadLink(link Link) core.Link {
	return core.Link{
		Source:       core.Identifier(link.Source),
		Destination:  core.Identifier(link.Destination),
		Relationship: link.Relationship,
	}
}
