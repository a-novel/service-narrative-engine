package core

import (
	"time"

	"github.com/a-novel-kit/golib/picture"
)

const (
	// ProjectLogoHeightMax caps the height of a logo, in pixels. Its width follows its proportions.
	ProjectLogoHeightMax = 128
	// ProjectLogoBytesMax caps an encoded logo, as the schema does.
	ProjectLogoBytesMax = 256 << 10
)

// projectLogoRule is what a logo accepts. A logo takes PNG because Safari cannot encode WebP and JPEG loses
// transparency.
var projectLogoRule = picture.Rule{MaxHeight: ProjectLogoHeightMax, MaxBytes: ProjectLogoBytesMax, AllowPNG: true}

// ProjectLogo is the logo of a project.
type ProjectLogo struct {
	// MediaType is the media type the logo is encoded in: image/webp, image/jpeg or image/png.
	MediaType string
	// Data holds the encoded image.
	Data []byte
	// Checksum is the SHA-256 of Data in lowercase hex, as [Project.LogoChecksum] carries it.
	Checksum string

	UpdatedAt time.Time
}
