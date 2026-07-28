package core

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrIdeaNotFound is returned when the actor cannot access the requested Idea.
var ErrIdeaNotFound = errors.New("idea not found")

// Idea is the typed entry contract from which an Engine Version generates content.
type Idea struct {
	ID        uuid.UUID  `json:"id"`
	OwnerID   uuid.UUID  `json:"ownerID"`
	Seed      string     `json:"seed"`
	Genre     string     `json:"genre"`
	Title     string     `json:"title"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}
