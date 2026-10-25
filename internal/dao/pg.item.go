package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Item is a placeholder resource stored in the database.
type Item struct {
	bun.BaseModel `bun:"table:items,alias:items"`

	ID          uuid.UUID `bun:"id,pk,type:uuid"`
	Name        string    `bun:"name"`
	Description string    `bun:"description"`
	CreatedAt   time.Time `bun:"created_at"`
	UpdatedAt   time.Time `bun:"updated_at"`
}
