package services

import (
	"time"

	"github.com/google/uuid"
)

// Item is the placeholder resource type.
type Item struct {
	ID          uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
