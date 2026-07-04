package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Item is the JSON representation of a narrative item returned by the REST API.
type Item struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// loadItem maps a core item onto its REST representation.
func loadItem(s *core.Item) Item {
	return Item{
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

// loadItemMap adapts loadItem to the signature lo.Map expects, ignoring the slice index.
func loadItemMap(item *core.Item, _ int) Item {
	return loadItem(item)
}
