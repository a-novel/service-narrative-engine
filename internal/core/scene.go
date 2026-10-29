package core

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// Scene is one subcolumn of the Scenes Composer: a part of a beat. Its fields describe the subcolumn;
// what happens in the scene lives in the cells of its layers.
type Scene struct {
	ID uuid.UUID
	// ProjectID is the project the scene belongs to.
	ProjectID uuid.UUID
	// BeatID is the beat the scene is part of.
	BeatID uuid.UUID

	// Position orders the beat's scenes, from 0.
	Position int

	// Title names the scene in the composer.
	Title string
	// Description refines the beat's description for the scene. Empty when the beat's is enough.
	Description string
	// Prompt refines the beat's prompt for the scene. Empty when the beat's is enough.
	Prompt string

	UpdatedAt time.Time
}

func newScene(entity *dao.Scene) *Scene {
	return &Scene{
		ID:          entity.ID,
		ProjectID:   entity.ProjectID,
		BeatID:      entity.BeatID,
		Position:    entity.Position,
		Title:       entity.Title,
		Description: entity.Description,
		Prompt:      entity.Prompt,
		UpdatedAt:   entity.UpdatedAt,
	}
}
