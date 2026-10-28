package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Project is a narrative in progress, owned by one user.
type Project struct {
	bun.BaseModel `bun:"table:projects,alias:projects"`

	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// OwnerID is the user who owns the project, from the authentication service.
	OwnerID uuid.UUID `bun:"owner_id,type:uuid"`

	// Title names the narrative.
	Title string `bun:"title"`
	// Premise says what the narrative is about, in a few sentences. Empty until the author writes one.
	Premise string `bun:"premise"`
	// Language is the language the project is written in: en or fr.
	Language string `bun:"language"`

	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}
