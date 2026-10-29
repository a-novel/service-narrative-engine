package core

import (
	"time"

	"github.com/google/uuid"
)

// ProjectSnapshot is a project with every scene, layer, cell and recap it holds, read at one instant. A
// generation reads its context from it.
type ProjectSnapshot struct {
	Project *Project `validate:"required"`

	// Scenes lists every scene of the project.
	Scenes []*Scene
	// Layers lists every layer of the project.
	Layers []*Layer
	// Cells lists every cell of the project that holds content.
	Cells []*Cell
	// Recaps lists the recaps of the project's cells.
	Recaps []*CellRecap
}

// CellRecap summarizes the content of one cell. A context that cannot hold a manuscript cell whole shows
// its recap instead, while the recap is fresh.
type CellRecap struct {
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID

	// Summary tells what happens in the cell, in about 80 words.
	Summary string
	// Names lists the characters, places and objects the cell names.
	Names []string

	// ContentUpdatedAt is when the content the recap read was written. The recap is stale once the cell
	// changes after that.
	ContentUpdatedAt time.Time
}
