package core

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ProjectSnapshot is what a generation plans from first: a project with every beat, scene and layer it
// holds, and a summary of each cell. The plan then reads the content and the recaps of the cells it shows.
type ProjectSnapshot struct {
	Project *Project `validate:"required"`

	// Beats lists every beat of the project.
	Beats []*Beat
	// Scenes lists every scene of the project.
	Scenes []*Scene
	// Layers lists every layer of the project.
	Layers []*Layer
	// Cells summarizes every cell of the project.
	Cells []*CellSummary
	// BeatRecaps lists the recaps of the project's beats.
	BeatRecaps []*BeatRecap
}

// CellSummary is what a plan decides on before it reads a cell: whether the cell holds content, how much,
// the speakers it names, and its recap without the versions.
type CellSummary struct {
	// SceneID is the scene whose column holds the cell.
	SceneID uuid.UUID
	// LayerID is the layer whose row holds the cell.
	LayerID uuid.UUID

	UpdatedAt time.Time

	// Empty marks a cell that holds no text and no block.
	Empty bool
	// Size counts the characters of every text, speaker and tone of the cell, as a context shows them
	// before any markup.
	Size int
	// Speakers lists the speakers of the cell's blocks, each once.
	Speakers []string

	// Recap summarizes the cell's latest recap, nil until one is written.
	Recap *RecapSummary
}

func newCellSummary(entity *dao.CellSummaryListRow) *CellSummary {
	summary := &CellSummary{
		SceneID: entity.SceneID, LayerID: entity.LayerID, UpdatedAt: entity.UpdatedAt,
		Empty: entity.Empty, Size: entity.Size, Speakers: entity.Speakers,
	}

	if entity.RecapCellUpdatedAt != nil {
		summary.Recap = &RecapSummary{
			Names:          entity.RecapNames,
			CondensedSize:  lo.FromPtr(entity.RecapCondensedSize),
			CompactSize:    lo.FromPtr(entity.RecapCompactSize),
			EssentialsSize: lo.FromPtr(entity.RecapEssentialsSize),
			CellUpdatedAt:  *entity.RecapCellUpdatedAt,
		}
	}

	return summary
}

// RecapSummary is a cell's recap without its versions. The recap is fresh while CellUpdatedAt equals the
// cell's UpdatedAt.
type RecapSummary struct {
	// Names lists the characters, places and objects the source names.
	Names []string
	// CondensedSize, CompactSize and EssentialsSize count the characters of each version.
	CondensedSize  int
	CompactSize    int
	EssentialsSize int

	// CellUpdatedAt is the cell's UpdatedAt when the recap read it.
	CellUpdatedAt time.Time
}

// size returns the characters of the recap's version at a level, as [Recap.version] picks it.
func (recap *RecapSummary) size(level string) int {
	switch level {
	case ContextLevelCondensed:
		return recap.CondensedSize
	case ContextLevelCompact:
		return recap.CompactSize
	default:
		return recap.EssentialsSize
	}
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

// newRecap returns nil for a source without a recap.
func newRecap(entity *dao.Recap) *Recap {
	if entity == nil {
		return nil
	}

	return &Recap{
		Condensed: entity.Condensed, Compact: entity.Compact, Essentials: entity.Essentials, Names: entity.Names,
	}
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
