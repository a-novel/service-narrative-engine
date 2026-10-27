package dao

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Cell is the content of one layer in one scene.
type Cell struct {
	bun.BaseModel `bun:"table:cells,alias:cells"`

	SceneID uuid.UUID `bun:"scene_id,pk,type:uuid"`
	LayerID uuid.UUID `bun:"layer_id,pk,type:uuid"`
	// Content is a JSON document that follows the content schema of the layer's kind.
	Content   json.RawMessage `bun:"content,type:jsonb"`
	UpdatedAt time.Time       `bun:"updated_at"`
}
