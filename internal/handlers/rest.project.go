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

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func loadProject(project *core.Project) Project {
	return Project{
		ID:        project.ID,
		Title:     project.Title,
		Premise:   project.Premise,
		Language:  project.Language,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
	}
}
