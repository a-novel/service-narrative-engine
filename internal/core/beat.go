package core

import (
	"time"

	"github.com/google/uuid"

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
	}
}
