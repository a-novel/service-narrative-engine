package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ProjectPicture is a picture a project holds, with its version.
type ProjectPicture struct {
	bun.BaseModel `bun:"table:project_pictures,alias:project_pictures"`

	ProjectID uuid.UUID `bun:"project_id,pk,type:uuid"`
	// Picture names the picture: logo or background.
	Picture string `bun:"picture,pk"`

	// Data holds the encoded image.
	Data []byte `bun:"data"`
	// MediaType is the media type the picture is encoded in, image/webp or image/jpeg.
	MediaType string `bun:"media_type"`

	// UpdatedAt is when the picture last changed, from the project's version column of the picture. A read
	// fills it, and an insert of the model leaves it out.
	UpdatedAt time.Time `bun:"updated_at,scanonly"`
}
