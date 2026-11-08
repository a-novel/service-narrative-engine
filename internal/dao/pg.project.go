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
	// UpdatedAt is when the project's own fields last changed.
	UpdatedAt time.Time `bun:"updated_at"`
	// EditedAt is when the project's content last changed: the instant the write that changed it locked the
	// project. A picture write leaves it as it was.
	EditedAt time.Time `bun:"edited_at"`

	// LogoUpdatedAt is when the logo last changed, nil while the project has no logo. A read joins it from the
	// logo's row, and an insert of the model leaves it out.
	LogoUpdatedAt *time.Time `bun:"logo_updated_at,scanonly"`
	// BackgroundUpdatedAt is when the background last changed, nil while the project has no background. A read
	// joins it from the background's row, and an insert of the model leaves it out.
	BackgroundUpdatedAt *time.Time `bun:"background_updated_at,scanonly"`
}
