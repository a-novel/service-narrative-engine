package core

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

const (
	// LayerKindManuscript is the kind of the permanent layer whose cells the manuscript exports.
	LayerKindManuscript = "manuscript"
	// LayerKindNotes is the kind of a layer of free text.
	LayerKindNotes = "notes"
)

// Layer is one row of the Scenes Composer: one kind of information followed across scenes.
type Layer struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	// Position orders the project's layers, from 0.
	Position int
	// Kind names the content schema the layer's cells follow.
	Kind        string
	Name        string
	Description string
	// Revision counts the writes to the layer's own fields. A move leaves it unchanged.
	Revision  int
	UpdatedAt time.Time
}

func newLayer(entity *dao.Layer, _ int) *Layer {
	return &Layer{
		ID:          entity.ID,
		ProjectID:   entity.ProjectID,
		Position:    entity.Position,
		Kind:        entity.Kind,
		Name:        entity.Name,
		Description: entity.Description,
		Revision:    entity.Revision,
		UpdatedAt:   entity.UpdatedAt,
	}
}
