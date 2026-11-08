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

	CreatedAt time.Time
	// UpdatedAt is when the project's own fields last changed: its title, premise or language.
	UpdatedAt time.Time
	// EditedAt is when the project's content last changed: its own fields, its beats, scenes, layers, cells or
	// recaps. Its pictures are cosmetic and leave it as it was.
	EditedAt time.Time

	// LogoVersion is when the logo last changed, nil while the project has no logo. Picture URLs carry it,
	// so a cache keeps each version for good.
	LogoVersion *time.Time
	// BackgroundVersion is when the background last changed, nil while the project has no background.
	BackgroundVersion *time.Time
}

func newProject(entity *dao.Project) *Project {
	return &Project{
		ID:                entity.ID,
		OwnerID:           entity.OwnerID,
		Title:             entity.Title,
		Premise:           entity.Premise,
		Language:          entity.Language,
		CreatedAt:         entity.CreatedAt,
		UpdatedAt:         entity.UpdatedAt,
		EditedAt:          entity.EditedAt,
		LogoVersion:       entity.LogoUpdatedAt,
		BackgroundVersion: entity.BackgroundUpdatedAt,
	}
}
