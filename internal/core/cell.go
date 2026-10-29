package core

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// SceneContentMax caps the characters all the cells of a scene hold together, counted over their stored
// JSON documents. That is about 100K tokens, so a generation that reads a whole scene fits one regular
// model call.
const SceneContentMax = 400000

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

	// Recap is the cell's latest recap, nil until one is written. It is fresh while RecapCellUpdatedAt
	// equals UpdatedAt.
	Recap *Recap
	// RecapCellUpdatedAt is the cell's UpdatedAt the recap read. It is zero without a recap.
	RecapCellUpdatedAt time.Time
}

func newCell(entity *dao.Cell) *Cell {
	return &Cell{
		ProjectID: entity.ProjectID,
		SceneID:   entity.SceneID,
		LayerID:   entity.LayerID,
		Content:   entity.Content,
		UpdatedAt: entity.UpdatedAt,

		Recap:              newRecap(entity.Recap),
		RecapCellUpdatedAt: lo.FromPtr(entity.RecapCellUpdatedAt),
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
