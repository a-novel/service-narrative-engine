package core

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// Beat is one column of the Scenes Composer: a turn of the story's structure, split into scenes. Its fields
// describe the column; what happens in it lives in the cells of its scenes.
type Beat struct {
	ID uuid.UUID
	// ProjectID is the project the beat belongs to.
	ProjectID uuid.UUID

	// Position orders the project's beats, from 0.
	Position int

	// Title names the beat in the composer.
	Title string
	// Color is the beat's color in the composer, as a lowercase #rrggbb hex code.
	Color string
	// Description says what the beat is for, written for the author.
	Description string
	// Prompt holds extra instructions for the model when it writes about the beat.
	Prompt string

	UpdatedAt time.Time

	// Recap is the beat's latest recap, nil until one is written. It is fresh while it read the beat, its
	// scenes and their manuscript cells as they are.
	Recap *Recap
	// RecapBeatUpdatedAt is the beat's UpdatedAt the recap read. It is zero without a recap.
	RecapBeatUpdatedAt time.Time
	// RecapSources lists, for each scene of the beat in order, the manuscript cell the recap read.
	RecapSources []*BeatRecapSource
}

func newBeat(entity *dao.Beat) *Beat {
	return &Beat{
		ID:          entity.ID,
		ProjectID:   entity.ProjectID,
		Position:    entity.Position,
		Title:       entity.Title,
		Color:       entity.Color,
		Description: entity.Description,
		Prompt:      entity.Prompt,
		UpdatedAt:   entity.UpdatedAt,

		Recap:              newRecap(entity.Recap),
		RecapBeatUpdatedAt: lo.FromPtr(entity.RecapBeatUpdatedAt),
		RecapSources:       newRecapSources(entity.RecapSources),
	}
}

// newRecapSources returns nil for a beat without a recap.
func newRecapSources(entities []*dao.BeatRecapSource) []*BeatRecapSource {
	if entities == nil {
		return nil
	}

	return lo.Map(entities, func(source *dao.BeatRecapSource, _ int) *BeatRecapSource {
		return &BeatRecapSource{SceneID: source.SceneID, CellUpdatedAt: source.CellUpdatedAt}
	})
}
