package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Project is the REST representation of a project.
type Project struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Premise   string    `json:"premise"`
	Language  string    `json:"language"`
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

// loadProjectMap adapts loadProject to lo.Map, ignoring the slice index.
func loadProjectMap(project *core.Project, _ int) Project {
	return loadProject(project)
}
