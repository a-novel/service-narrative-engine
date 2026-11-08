package core

import (
	"time"

	"github.com/a-novel-kit/golib/picture"
)

const (
	// ProjectPictureLogo names the logo of a project, a square.
	ProjectPictureLogo = "logo"
	// ProjectPictureBackground names the background of a project, a 16:9 picture.
	ProjectPictureBackground = "background"
)

const (
	// ProjectLogoWidthMax caps the width of a logo, in pixels: the platform draws it 512 px wide at most.
	ProjectLogoWidthMax = 512
	// ProjectLogoBytesMax caps an encoded logo, as the schema does.
	ProjectLogoBytesMax = 256 << 10
	// ProjectBackgroundWidthMax caps the width of a background, in pixels: the platform draws it 1280 px
	// wide at most.
	ProjectBackgroundWidthMax = 1280
	// ProjectBackgroundBytesMax caps an encoded background, as the schema does.
	ProjectBackgroundBytesMax = 512 << 10

	// backgroundRatioWidth and backgroundRatioHeight set the aspect ratio of a background, 16:9.
	backgroundRatioWidth, backgroundRatioHeight = 16, 9
)

// projectPictureContract is what a picture of a project accepts.
type projectPictureContract struct {
	rule picture.Rule
	// maxBytes caps the encoded picture.
	maxBytes int
}

// projectPictureContracts maps each picture a project may hold to what it accepts. The schema's
// project_picture enum holds the same names.
var projectPictureContracts = map[string]projectPictureContract{
	ProjectPictureLogo: {
		rule:     picture.Rule{RatioWidth: 1, RatioHeight: 1, MaxWidth: ProjectLogoWidthMax},
		maxBytes: ProjectLogoBytesMax,
	},
	ProjectPictureBackground: {
		rule: picture.Rule{
			RatioWidth: backgroundRatioWidth, RatioHeight: backgroundRatioHeight, MaxWidth: ProjectBackgroundWidthMax,
		},
		maxBytes: ProjectBackgroundBytesMax,
	},
}

// ProjectPicture is a picture of a project.
type ProjectPicture struct {
	// MediaType is the media type the picture is encoded in, image/webp or image/jpeg.
	MediaType string
	// Data holds the encoded image.
	Data []byte

	// Version is when the picture last changed.
	Version time.Time
}
