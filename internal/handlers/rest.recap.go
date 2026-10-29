package handlers

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Recap condenses a cell or a beat at three levels of detail, as a consolidation wrote it.
type Recap struct {
	// Condensed holds about half of the source.
	Condensed string `json:"condensed"`
	// Compact holds about a tenth of the source.
	Compact string `json:"compact"`
	// Essentials holds the source in one sentence.
	Essentials string `json:"essentials"`

	// Names lists the characters, places and objects the source names.
	Names []string `json:"names"`
}

// BeatRecapSource is one scene a beat's recap read, with the updatedAt of its manuscript cell.
type BeatRecapSource struct {
	SceneID uuid.UUID `json:"sceneID"`
	// CellUpdatedAt is the updatedAt of the scene's manuscript cell the recap read, or null for a scene
	// without manuscript.
	CellUpdatedAt *time.Time `json:"cellUpdatedAt"`
}

// loadRecap returns nil for a source without a recap.
func loadRecap(recap *core.Recap) *Recap {
	if recap == nil {
		return nil
	}

	return &Recap{Condensed: recap.Condensed, Compact: recap.Compact, Essentials: recap.Essentials, Names: recap.Names}
}

// loadBeatRecapSources returns nil for a beat without sources.
func loadBeatRecapSources(sources []*core.BeatRecapSource) []*BeatRecapSource {
	if sources == nil {
		return nil
	}

	return lo.Map(sources, func(source *core.BeatRecapSource, _ int) *BeatRecapSource {
		return &BeatRecapSource{SceneID: source.SceneID, CellUpdatedAt: optionalTime(source.CellUpdatedAt)}
	})
}

// optionalTime returns nil for a zero instant.
func optionalTime(instant time.Time) *time.Time {
	if instant.IsZero() {
		return nil
	}

	return &instant
}
