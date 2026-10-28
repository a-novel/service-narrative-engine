package core

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

const (
	// LayerKindManuscript is the kind of the layer whose cells the manuscript exports.
	LayerKindManuscript = "manuscript"
	// LayerKindNotes is the kind of a layer of free text.
	LayerKindNotes = "notes"
)

// requiredLayers maps the kind of each layer every project holds to the name a new project gives
// it. A project starts with these layers, by kind order, and none of them can be deleted. Clients
// show their own label for a required kind, in the reader's language.
var requiredLayers = map[string]string{
	LayerKindManuscript: "Manuscript",
}

// Layer is one row of the Scenes Composer: one kind of information followed across scenes.
type Layer struct {
	ID uuid.UUID
	// ProjectID is the project the layer belongs to.
	ProjectID uuid.UUID

	// Position orders the project's layers, from 0.
	Position int
	// Kind names the content schema the layer's cells follow.
	Kind string

	// Name labels the layer's row in the composer.
	Name string
	// Description says what the layer records.
	Description string

	UpdatedAt time.Time
}

func newLayer(entity *dao.Layer) *Layer {
	return &Layer{
		ID:          entity.ID,
		ProjectID:   entity.ProjectID,
		Position:    entity.Position,
		Kind:        entity.Kind,
		Name:        entity.Name,
		Description: entity.Description,
		UpdatedAt:   entity.UpdatedAt,
	}
}
