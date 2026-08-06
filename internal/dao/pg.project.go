package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Project is the stable identity and ownership root for story content.
type Project struct {
	bun.BaseModel `bun:"table:projects,alias:project"`

	// ID identifies the Project.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// OwnerID identifies the user who owns the Project.
	OwnerID uuid.UUID `bun:"owner_id,type:uuid"`
	// CreatedAt records when the Project was created.
	CreatedAt time.Time `bun:"created_at"`
}
