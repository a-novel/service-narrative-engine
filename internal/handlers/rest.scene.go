package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Scene is the REST representation of a scene.
type Scene struct {
	ID uuid.UUID `json:"id"`
	// ProjectID is the project the scene belongs to.
	ProjectID uuid.UUID `json:"projectID"`

	// Position orders the project's scenes, from 0.
	Position int `json:"position"`

	// Title names the scene in the composer.
	Title string `json:"title"`
	// Color is the scene's color in the composer, as a lowercase #rrggbb hex code.
	Color string `json:"color"`
	// Description says what the scene is for, written for the author.
	Description string `json:"description"`
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt string `json:"prompt"`

	UpdatedAt time.Time `json:"updatedAt"`
}

func loadScene(scene *core.Scene) Scene {
	return Scene{
		ID:          scene.ID,
		ProjectID:   scene.ProjectID,
		Position:    scene.Position,
		Title:       scene.Title,
		Color:       scene.Color,
		Description: scene.Description,
		Prompt:      scene.Prompt,
		UpdatedAt:   scene.UpdatedAt,
	}
}
