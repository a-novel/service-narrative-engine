package core

import (
	"time"

	"github.com/google/uuid"
)

// ProjectSnapshot is a project with every beat, scene, layer, cell and recap it holds, read at one instant.
// A generation reads its context from it.
type ProjectSnapshot struct {
	Project *Project `validate:"required"`

	// Beats lists every beat of the project.
	Beats []*Beat
	// Scenes lists every scene of the project.
	Scenes []*Scene
	// Layers lists every layer of the project.
	Layers []*Layer
	// Cells lists every cell of the project that holds content.
	Cells []*Cell
	// Recaps lists the recaps of the project's cells.
	Recaps []*CellRecap
	// BeatRecaps lists the recaps of the project's beats.
	BeatRecaps []*BeatRecap
}

// Recap condenses a source at three levels of detail, richest first, as a game keeps simpler models for far
// objects. A context shows the version a scene's distance and its budget allow.
type Recap struct {
	// Condensed holds about half of the source.
	Condensed string
	// Compact holds about a tenth of the source.
	Compact string
	// Essentials holds the source in one sentence.
	Essentials string

	// Names lists the characters, places and objects the source names.
	Names []string
}

// version returns the recap at a level: [ContextLevelCondensed], [ContextLevelCompact] or
// [ContextLevelEssentials].
func (recap *Recap) version(level string) string {
	switch level {
	case ContextLevelCondensed:
		return recap.Condensed
	case ContextLevelCompact:
		return recap.Compact
	default:
		return recap.Essentials
	}
}

// CellRecap recaps the content of one cell. A context that cannot hold a manuscript cell whole shows one of
// its versions instead, while the recap is fresh.
type CellRecap struct {
	Recap

	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID

	// CellUpdatedAt is the cell's UpdatedAt when the recap read it. The recap is stale once they differ.
	CellUpdatedAt time.Time
}

// BeatRecap recaps a beat: its description and the manuscript of its scenes. A frame that cannot list a
// far beat's scenes shows one of its versions instead, while the recap is fresh.
type BeatRecap struct {
	Recap

	BeatID uuid.UUID

	// BeatUpdatedAt is the beat's UpdatedAt when the recap read it.
	BeatUpdatedAt time.Time
	// Sources lists, for each scene of the beat in order, the manuscript cell the recap read. The recap
	// is stale once the beat, its scenes or one of those cells changed.
	Sources []*BeatRecapSource
}

// BeatRecapSource is one scene a [BeatRecap] read.
type BeatRecapSource struct {
	SceneID uuid.UUID
	// CellUpdatedAt is the UpdatedAt of the scene's manuscript cell when the recap read it. It is zero for
	// a scene without manuscript.
	CellUpdatedAt time.Time
}
