package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// ProjectFields is the REST representation of a project's fields, with its EditedAt so a project the client lacks
// arrives whole. It carries no logo.
type ProjectFields struct {
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
}

func loadProjectFields(project *core.Project) ProjectFields {
	return ProjectFields{
		ID:        project.ID,
		Title:     project.Title,
		Premise:   project.Premise,
		Language:  project.Language,
		Tags:      project.Tags,
		Genres:    project.Genres,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
		EditedAt:  project.EditedAt,
	}
}
