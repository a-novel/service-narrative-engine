package handlers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Cell is the REST representation of a cell.
type Cell struct {
	SceneID   uuid.UUID       `json:"sceneID"`
	LayerID   uuid.UUID       `json:"layerID"`
	Content   json.RawMessage `json:"content"`
	Revision  int             `json:"revision"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

func loadCell(cell *core.Cell) Cell {
	return Cell{
		SceneID:   cell.SceneID,
		LayerID:   cell.LayerID,
		Content:   cell.Content,
		Revision:  cell.Revision,
		UpdatedAt: cell.UpdatedAt,
	}
}

// loadCellMap adapts loadCell to lo.Map, ignoring the slice index.
func loadCellMap(cell *core.Cell, _ int) Cell {
	return loadCell(cell)
}
