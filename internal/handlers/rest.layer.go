package handlers

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Layer is the REST representation of a layer.
type Layer struct {
	ID uuid.UUID `json:"id"`
	// ProjectID is the project the layer belongs to.
	ProjectID uuid.UUID `json:"projectID"`

	// Position orders the project's layers, from 0.
	Position int `json:"position"`
	// Kind names the content schema the layer's cells follow.
	Kind string `json:"kind"`

	// Name labels the layer's row in the composer.
	Name string `json:"name"`
	// Description says what the layer records.
	Description string `json:"description"`

	UpdatedAt time.Time `json:"updatedAt"`
}

func loadLayer(layer *core.Layer) Layer {
	return Layer{
		ID:          layer.ID,
		ProjectID:   layer.ProjectID,
		Position:    layer.Position,
		Kind:        layer.Kind,
		Name:        layer.Name,
		Description: layer.Description,
		UpdatedAt:   layer.UpdatedAt,
	}
}

// LayerPage is the REST representation of a page of a project's layers.
type LayerPage struct {
	Layers []Layer `json:"layers"`

	// Total counts every layer of the project, so a client paging through them knows when it holds
	// them all.
	Total int `json:"total"`
}

func loadLayerPage(page *core.LayerPage) LayerPage {
	return LayerPage{
		Layers: lo.Map(page.Layers, func(layer *core.Layer, _ int) Layer { return loadLayer(layer) }),
		Total:  page.Total,
	}
}
