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

// Link is a static link: it relates an item of the instance that stores it, its importer, to an item of an
// instance of the same project, its exporter. Its JSON form is the one the instance stores, and the one a read
// returns.
type Link struct {
	// Source is the address of the importing item inside the instance that stores the link.
	Source string `json:"source" validate:"required"`
	// Destination is the exported item. Its instance may be the one that stores the link.
	Destination Identifier `json:"destination"`
	// Relationship says what the source is to the destination, in the words of the plugin that wrote the link.
	Relationship string `json:"relationship"`
}

// InstanceLinks is the static links an instance stores, which it replaces whole.
type InstanceLinks struct {
	// InstanceID is the instance that stores the links, their importer.
	InstanceID uuid.UUID

	// Links is a JSON array of [Link], empty for an instance that never wrote its links.
	Links json.RawMessage

	// UpdatedAt is when the links last changed, to the microsecond. A write names it. It is nil for an instance
	// that never wrote its links.
	UpdatedAt *time.Time
}

// InstanceLinksPage is a page of the links the instances of a project store.
type InstanceLinksPage struct {
	Instances []*InstanceLinks

	// Total counts every instance of the project, so a client paging through them knows when it holds the
	// links of them all. A page past the end reports 0.
	Total int
}

// WrittenInstanceLinks is when a write replaced the links of an instance. The caller already holds them.
type WrittenInstanceLinks struct {
	UpdatedAt time.Time
}
