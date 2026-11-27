package handlers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
)

// Identifier is the REST representation of the destination of a link: an item of a project.
type Identifier struct {
	// ID is the instance that holds the item.
	ID uuid.UUID `json:"id"`
	// Target is the address of the item inside that instance.
	Target string `json:"target"`
}

// Link is the REST representation of a static link, from an item of the instance that stores it to an item of
// an instance of the same project.
type Link struct {
	// Source is the address of the importing item inside the instance that stores the link.
	Source string `json:"source"`
	// Destination is the exported item.
	Destination Identifier `json:"destination"`
	// Relationship says what the source is to the destination, in the words of the plugin that wrote the link.
	Relationship string `json:"relationship"`
}

// InstanceLinks is the REST representation of the static links an instance stores.
type InstanceLinks struct {
	// InstanceID is the instance that stores the links, their importer.
	InstanceID uuid.UUID `json:"instanceID"`

	// Links is the list of [Link] as the instance stores it, which a read returns without decoding it. It is
	// empty for an instance that never wrote its links.
	Links json.RawMessage `json:"links"`

	// UpdatedAt is when the links last changed, null for an instance that never wrote its links. Its JSON keeps
	// the microseconds the service stores, and a write sends it back as it read it.
	UpdatedAt *time.Time `json:"updatedAt"`
}

// InstanceLinksPage is the REST representation of a page of the links the instances of a project store.
type InstanceLinksPage struct {
	Instances []InstanceLinks `json:"instances"`

	// Total counts every instance of the project, beyond the page.
	Total int `json:"total"`
}

// WrittenInstanceLinks is the REST representation of a write of the links, which the next write names.
type WrittenInstanceLinks struct {
	UpdatedAt time.Time `json:"updatedAt"`
}

func loadLink(link Link) core.Link {
	return core.Link{
		Source:       link.Source,
		Destination:  core.Identifier(link.Destination),
		Relationship: link.Relationship,
	}
}

func loadInstanceLinksPage(page *core.InstanceLinksPage) InstanceLinksPage {
	return InstanceLinksPage{
		Instances: lo.Map(page.Instances, func(links *core.InstanceLinks, _ int) InstanceLinks {
			return InstanceLinks{InstanceID: links.InstanceID, Links: links.Links, UpdatedAt: links.UpdatedAt}
		}),
		Total: page.Total,
	}
}
