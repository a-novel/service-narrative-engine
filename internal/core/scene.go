package core

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// Scene is one column of the Scenes Composer: a unit of narrative order, with a title and a
// synopsis.
type Scene struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	// Position orders the project's scenes, from 0.
	Position int
	Title    string
	// Synopsis is the writer's short plan of the scene.
	Synopsis string
	// Revision counts the writes to the scene's own fields. A move leaves it unchanged.
	Revision  int
	UpdatedAt time.Time
}

func newScene(entity *dao.Scene, _ int) *Scene {
	return &Scene{
		ID:        entity.ID,
		ProjectID: entity.ProjectID,
		Position:  entity.Position,
		Title:     entity.Title,
		Synopsis:  entity.Synopsis,
		Revision:  entity.Revision,
		UpdatedAt: entity.UpdatedAt,
	}
}
