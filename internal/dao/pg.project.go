package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Project is a narrative in progress, owned by one user.
type Project struct {
	bun.BaseModel `bun:"table:projects,alias:projects"`

	ID       uuid.UUID `bun:"id,pk,type:uuid"`
	OwnerID  uuid.UUID `bun:"owner_id,type:uuid"`
	Title    string    `bun:"title"`
	Premise  string    `bun:"premise"`
	Language string    `bun:"language"`
	// Revision counts the writes to the project's own fields.
	Revision  int       `bun:"revision"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}
