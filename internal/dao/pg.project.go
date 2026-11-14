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
	// Tags are the author's custom labels, in order. Empty until the author adds one.
	Tags []string `bun:"tags,array"`
	// Genres are the genres the author names, in order. Empty until the author adds one.
	Genres []string `bun:"genres,array"`

	CreatedAt time.Time `bun:"created_at"`
	// UpdatedAt is when the project's own fields last changed.
	UpdatedAt time.Time `bun:"updated_at"`
	// EditedAt is when the project's content last changed: the instant the write that changed it locked the
	// project. A logo write leaves it as it was.
	EditedAt time.Time `bun:"edited_at"`

	// LogoChecksum is the SHA-256 of the logo's bytes in lowercase hex, nil while the project has no logo. A read
	// joins it from the logo's row, and an insert of the model leaves it out.
	LogoChecksum *string `bun:"logo_checksum,scanonly"`
}
