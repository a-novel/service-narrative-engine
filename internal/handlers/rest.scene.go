package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Scene is the REST representation of a scene.
type Scene struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"projectID"`
	Position    int       `json:"position"`
	Title       string    `json:"title"`
	Color       string    `json:"color"`
	Description string    `json:"description"`
	Prompt      string    `json:"prompt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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

// loadSceneMap adapts loadScene to lo.Map, ignoring the slice index.
func loadSceneMap(scene *core.Scene, _ int) Scene {
	return loadScene(scene)
}
