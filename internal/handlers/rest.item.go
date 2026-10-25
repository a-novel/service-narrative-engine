package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Item is the REST representation of an item returned by the public endpoints.
type Item struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// loadItem converts a core item into its REST form.
func loadItem(item *core.Item) Item {
	return Item{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

// loadItemMap adapts loadItem to lo.Map, ignoring the slice index.
func loadItemMap(item *core.Item, _ int) Item {
	return loadItem(item)
}
