package core

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// Scene is one column of the Scenes Composer: a unit of narrative order. Its fields describe the
// column; what happens in the scene lives in the cells of its layers.
type Scene struct {
	ID uuid.UUID
	// ProjectID is the project the scene belongs to.
	ProjectID uuid.UUID

	// Position orders the project's scenes, from 0.
	Position int

	// Title names the scene in the composer.
	Title string
	// Color is the scene's color in the composer, as a lowercase #rrggbb hex code.
	Color string
	// Description says what the scene is for, written for the author.
	Description string
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt string

	UpdatedAt time.Time
}

func newScene(entity *dao.Scene) *Scene {
	return &Scene{
		ID:          entity.ID,
		ProjectID:   entity.ProjectID,
		Position:    entity.Position,
		Title:       entity.Title,
		Color:       entity.Color,
		Description: entity.Description,
		Prompt:      entity.Prompt,
		UpdatedAt:   entity.UpdatedAt,
	}
}
