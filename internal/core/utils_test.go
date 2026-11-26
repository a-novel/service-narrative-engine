package core_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

var (
	ownerID   = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	projectID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

	createdAt = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt = time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)
	editedAt  = time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)
)

// logoChecksum is the checksum of the test project's logo: a SHA-256 in lowercase hex.
const logoChecksum = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

// daoMock is the answer a DAO mock gives. Go allows no generic type inside a function, so the
// tests share this one.
type daoMock[Resp any] struct {
	resp Resp
	err  error
}

// recent reports whether a request's instant was captured during the test.
func recent(instant time.Time) bool {
	return time.Since(instant) < time.Minute
}

// lockRequest matches the lock of the test project by its owner, which stamps the edit at an instant the
// service captures.
var lockRequest = mock.MatchedBy(func(request *dao.ProjectLockRequest) bool {
	return request.ID == projectID && request.OwnerID == ownerID && recent(request.Now)
})

// tenLabels fills a project's tags or genres.
var tenLabels = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}

func daoProject() *dao.Project {
	return &dao.Project{
		ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr", Tags: []string{"noir"},
		Genres:    []string{"thriller"},
		CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: editedAt,
		LogoChecksum: new(logoChecksum),
	}
}

// coreProject returns the project [daoProject] reads.
func coreProject() *core.Project {
	return &core.Project{
		ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr", Tags: []string{"noir"},
		Genres:    []string{"thriller"},
		CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: editedAt,
		LogoChecksum: new(logoChecksum),
	}
}

// newPNG encodes a black image of the given size as a PNG.
func newPNG(width, height int) []byte {
	var buf bytes.Buffer

	err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, width, height)))
	if err != nil {
		panic(err)
	}

	return buf.Bytes()
}

// newJPEG encodes a gray gradient of the given size as a JPEG.
func newJPEG(width, height int) []byte {
	img := image.NewGray(image.Rect(0, 0, width, height))

	for x := range width {
		for y := range height {
			img.SetGray(x, y, color.Gray{Y: uint8((x + y) % 256)})
		}
	}

	var buf bytes.Buffer

	err := jpeg.Encode(&buf, img, nil)
	if err != nil {
		panic(err)
	}

	return buf.Bytes()
}

var (
	// withinBudget is a project at its byte budget, which a write may reach.
	withinBudget = &daoMock[*dao.ProjectBytes]{resp: &dao.ProjectBytes{StoredBytes: core.ProjectBytesMax}}
	// overBudget is a project one byte past its budget, which refuses the write that brought it there.
	overBudget = &daoMock[*dao.ProjectBytes]{resp: &dao.ProjectBytes{StoredBytes: core.ProjectBytesMax + 1}}
)
