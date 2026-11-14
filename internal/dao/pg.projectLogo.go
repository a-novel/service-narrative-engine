package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ProjectLogo is the logo a project holds.
type ProjectLogo struct {
	bun.BaseModel `bun:"table:project_logos,alias:project_logos"`

	ProjectID uuid.UUID `bun:"project_id,pk,type:uuid"`

	// Data holds the encoded image.
	Data []byte `bun:"data"`
	// MediaType is the media type the logo is encoded in: image/webp, image/jpeg or image/png.
	MediaType string `bun:"media_type"`
	// Checksum is the SHA-256 of Data in lowercase hex. The database generates it, so an insert of the model
	// leaves it out.
	Checksum string `bun:"checksum,scanonly"`

	// UpdatedAt is when the logo last changed.
	UpdatedAt time.Time `bun:"updated_at"`
}
