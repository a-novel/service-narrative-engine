package core

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-template/internal/dao"
)

// Item is the placeholder resource type.
type Item struct {
	ID          uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func newItem(entity *dao.Item) *Item {
	return &Item{
		ID:          entity.ID,
		Name:        entity.Name,
		Description: entity.Description,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}
}
