package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-template/internal/services"
)

type Item struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func loadItem(s *services.Item) Item {
	return Item{
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func loadItemMap(item *services.Item, _ int) Item {
	return loadItem(item)
}
