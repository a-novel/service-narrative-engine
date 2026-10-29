package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Beat is one column of a project's Scenes Composer: a unit of narrative order.
type Beat struct {
	bun.BaseModel `bun:"table:beats,alias:beats"`

	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// ProjectID is the project the beat belongs to.
	ProjectID uuid.UUID `bun:"project_id,type:uuid"`

	// Position orders the project's beats, from 0.
	Position int `bun:"position"`

	// Title names the beat in the composer.
	Title string `bun:"title"`
	// Color is the beat's color in the composer, as a lowercase #rrggbb hex code.
	Color string `bun:"color"`
	// Description says what the beat is for, written for the author.
	Description string `bun:"description"`
	// Prompt holds extra instructions for the model when it writes about the beat.
	Prompt string `bun:"prompt"`

	UpdatedAt time.Time `bun:"updated_at"`

	// Recap is the beat's latest recap. It is nil until one is written.
	Recap *Recap `bun:"recap,type:jsonb,nullzero"`
	// RecapBeatUpdatedAt is the beat's UpdatedAt the recap read. It is nil without a recap.
	RecapBeatUpdatedAt *time.Time `bun:"recap_beat_updated_at,nullzero"`
	// RecapSources lists, for each scene of the beat in order, the manuscript cell the recap read.
	RecapSources []*BeatRecapSource `bun:"recap_sources,type:jsonb,nullzero"`
}

// BeatRecapSource is one scene a beat's recap read.
type BeatRecapSource struct {
	SceneID uuid.UUID `json:"sceneID"`
	// CellUpdatedAt is the UpdatedAt of the scene's manuscript cell the recap read. It is zero for a scene
	// without manuscript.
	CellUpdatedAt time.Time `json:"cellUpdatedAt"`
}
