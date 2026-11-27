package core

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Identifier addresses one item of a project.
type Identifier struct {
	// ID is the instance that holds the item. The service never checks that it exists.
	ID uuid.UUID `json:"id" validate:"required"`
	// Target is the address of the item inside that instance, which only plugins read.
	Target string `json:"target" validate:"required"`
}

// Link relates two items of one project. Its JSON form is the one the project stores, and the one a read
// returns.
type Link struct {
	Source      Identifier `json:"source"`
	Destination Identifier `json:"destination"`
	// Relationship says what the source is to the destination, in the words of the plugin that wrote the link.
	Relationship string `json:"relationship"`
}

// ProjectLinks is the links of a project, which it stores and replaces whole.
type ProjectLinks struct {
	// Links is a JSON array of [Link].
	Links json.RawMessage

	// UpdatedAt is when the links last changed, to the microsecond. A write names it. It is nil for a project
	// that never wrote its links.
	UpdatedAt *time.Time
}

// WrittenProjectLinks is when a write replaced the links of a project. The caller already holds them.
type WrittenProjectLinks struct {
	UpdatedAt time.Time
}
