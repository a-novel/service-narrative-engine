package handlers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Cell is the REST representation of a cell.
type Cell struct {
	// ProjectID is the project the cell's scene and layer belong to.
	ProjectID uuid.UUID `json:"projectID"`
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID `json:"sceneID"`
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID `json:"layerID"`

	// Content is a JSON document that follows the content schema of the layer's kind.
	Content json.RawMessage `json:"content"`

	UpdatedAt time.Time `json:"updatedAt"`
}

func loadCell(cell *core.Cell) Cell {
	return Cell{
		ProjectID: cell.ProjectID,
		SceneID:   cell.SceneID,
		LayerID:   cell.LayerID,
		Content:   cell.Content,
		UpdatedAt: cell.UpdatedAt,
	}
}
