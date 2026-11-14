package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Project is the REST representation of a project.
type Project struct {
	ID uuid.UUID `json:"id"`

	// Title names the narrative.
	Title string `json:"title"`
	// Premise says what the narrative is about, in a few sentences.
	Premise string `json:"premise"`
	// Language is the language the project is written in, one of [core.ProjectLanguages].
	Language string `json:"language"`
	// Tags are the author's custom labels, in order. Empty until the author adds one.
	Tags []string `json:"tags"`
	// Genres are the genres the author names, in order. Empty until the author adds one.
	Genres []string `json:"genres"`

	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is when the project's own fields last changed.
	UpdatedAt time.Time `json:"updatedAt"`
	// EditedAt is when the project's content last changed, its logo aside.
	EditedAt time.Time `json:"editedAt"`

	// LogoChecksum is the SHA-256 of the logo's bytes in lowercase hex, absent while the project has no logo.
	LogoChecksum *string `json:"logoChecksum,omitempty"`
}

func loadProject(project *core.Project) Project {
	return Project{
		ID:           project.ID,
		Title:        project.Title,
		Premise:      project.Premise,
		Language:     project.Language,
		Tags:         project.Tags,
		Genres:       project.Genres,
		CreatedAt:    project.CreatedAt,
		UpdatedAt:    project.UpdatedAt,
		EditedAt:     project.EditedAt,
		LogoChecksum: project.LogoChecksum,
	}
}
