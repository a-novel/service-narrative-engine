package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Layer is the REST representation of a layer.
type Layer struct {
	ID          uuid.UUID `json:"id"`
	ProjectID   uuid.UUID `json:"projectID"`
	Position    int       `json:"position"`
	Kind        string    `json:"kind"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Revision    int       `json:"revision"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func loadLayer(layer *core.Layer) Layer {
	return Layer{
		ID:          layer.ID,
		ProjectID:   layer.ProjectID,
		Position:    layer.Position,
		Kind:        layer.Kind,
		Name:        layer.Name,
		Description: layer.Description,
		Revision:    layer.Revision,
		UpdatedAt:   layer.UpdatedAt,
	}
}

// loadLayerMap adapts loadLayer to lo.Map, ignoring the slice index.
func loadLayerMap(layer *core.Layer, _ int) Layer {
	return loadLayer(layer)
}
