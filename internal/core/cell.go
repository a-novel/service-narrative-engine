package core

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// Cell is the content of one layer in one scene.
type Cell struct {
	// ProjectID is the project the cell's scene and layer belong to.
	ProjectID uuid.UUID
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID

	// Content is a JSON document that follows the content schema of the layer's kind.
	Content json.RawMessage

	UpdatedAt time.Time
}

func newCell(entity *dao.Cell) *Cell {
	return &Cell{
		ProjectID: entity.ProjectID,
		SceneID:   entity.SceneID,
		LayerID:   entity.LayerID,
		Content:   entity.Content,
		UpdatedAt: entity.UpdatedAt,
	}
}

// WrittenCell names a cell [CellWrite] wrote, and when. The caller already holds its content.
type WrittenCell struct {
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID

	UpdatedAt time.Time
}

func newWrittenCell(row *dao.CellUpsertRow) *WrittenCell {
	return &WrittenCell{SceneID: row.SceneID, LayerID: row.LayerID, UpdatedAt: row.UpdatedAt}
}
