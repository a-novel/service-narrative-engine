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
	// UpdatedAt is when the project's own fields last changed.
	UpdatedAt time.Time `json:"updatedAt"`
	// EditedAt is when anything in the project last changed.
	EditedAt time.Time `json:"editedAt"`

	// LogoVersion is when the logo last changed, absent while the project has no logo.
	LogoVersion *time.Time `json:"logoVersion,omitempty"`
	// BackgroundVersion is when the background last changed, absent while the project has no background.
	BackgroundVersion *time.Time `json:"backgroundVersion,omitempty"`
}

func loadProject(project *core.Project) Project {
	return Project{
		ID:                project.ID,
		Title:             project.Title,
		Premise:           project.Premise,
		Language:          project.Language,
		CreatedAt:         project.CreatedAt,
		UpdatedAt:         project.UpdatedAt,
		EditedAt:          project.EditedAt,
		LogoVersion:       project.LogoVersion,
		BackgroundVersion: project.BackgroundVersion,
	}
}
