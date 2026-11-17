package handlers

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Scene is the REST representation of a scene.
type Scene struct {
	ID uuid.UUID `json:"id"`
	// ProjectID is the project the scene belongs to.
	ProjectID uuid.UUID `json:"projectID"`
	// BeatID is the beat the scene is part of.
	BeatID uuid.UUID `json:"beatID"`

	// Position orders the beat's scenes, from 0.
	Position int `json:"position"`

	// Title names the scene in the composer.
	Title string `json:"title"`
	// Description refines the beat's description for the scene. Empty when the beat's is enough.
	Description string `json:"description"`
	// Prompt refines the beat's prompt for the scene. Empty when the beat's is enough.
	Prompt string `json:"prompt"`

	UpdatedAt time.Time `json:"updatedAt"`
}

func loadScene(scene *core.Scene) Scene {
	return Scene{
		ID:          scene.ID,
		ProjectID:   scene.ProjectID,
		BeatID:      scene.BeatID,
		Position:    scene.Position,
		Title:       scene.Title,
		Description: scene.Description,
		Prompt:      scene.Prompt,
		UpdatedAt:   scene.UpdatedAt,
	}
}

// ScenePage is the REST representation of a page of a project's scenes.
type ScenePage struct {
	Scenes []Scene `json:"scenes"`

	// Total counts every scene of the project, so a client paging through them knows when it holds
	// them all.
	Total int `json:"total"`
}

func loadScenePage(page *core.ScenePage) ScenePage {
	return ScenePage{
		Scenes: lo.Map(page.Scenes, func(scene *core.Scene, _ int) Scene { return loadScene(scene) }),
		Total:  page.Total,
	}
}
