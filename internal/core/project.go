package core

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ProjectLanguages lists the languages a project can be written in, as BCP 47 tags. The schema's
// project_language enum holds the same values.
var ProjectLanguages = []string{"en", "fr"}

// Project is a narrative in progress, owned by one user. It produces one manuscript.
type Project struct {
	ID uuid.UUID
	// OwnerID is the user who owns the project, from the authentication service.
	OwnerID uuid.UUID

	// Title names the narrative.
	Title string
	// Premise says what the narrative is about, in a few sentences. Empty until the author writes one.
	Premise string
	// Language is the language the project is written in, as a BCP 47 tag.
	Language string
	// Tags are the author's custom labels, in order. Empty until the author adds one.
	Tags []string
	// Genres are the genres the author names, in order. Empty until the author adds one.
	Genres []string

	CreatedAt time.Time
	// UpdatedAt is when the project's own fields last changed: its title, premise, language, tags or genres.
	UpdatedAt time.Time
	// EditedAt is when the project's content last changed: its own fields, its beats, scenes, layers, cells or
	// recaps. Its logo is cosmetic and leaves it as it was.
	EditedAt time.Time

	// LogoChecksum is the SHA-256 of the logo's bytes in lowercase hex, nil while the project has no logo. Logo URLs
	// carry it, so a cache keeps each logo for good.
	LogoChecksum *string
}

func newProject(entity *dao.Project) *Project {
	return &Project{
		ID:           entity.ID,
		OwnerID:      entity.OwnerID,
		Title:        entity.Title,
		Premise:      entity.Premise,
		Language:     entity.Language,
		Tags:         entity.Tags,
		Genres:       entity.Genres,
		CreatedAt:    entity.CreatedAt,
		UpdatedAt:    entity.UpdatedAt,
		EditedAt:     entity.EditedAt,
		LogoChecksum: entity.LogoChecksum,
	}
}
