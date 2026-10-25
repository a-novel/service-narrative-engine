package core

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// Cell is the content of one layer in one scene.
type Cell struct {
	SceneID uuid.UUID
	LayerID uuid.UUID
	// Content is a JSON document that follows the content schema of the layer's kind.
	Content json.RawMessage
	// Revision counts the writes to the cell.
	Revision  int
	UpdatedAt time.Time
}

func newCell(entity *dao.Cell, _ int) *Cell {
	return &Cell{
		SceneID:   entity.SceneID,
		LayerID:   entity.LayerID,
		Content:   entity.Content,
		Revision:  entity.Revision,
		UpdatedAt: entity.UpdatedAt,
	}
}
