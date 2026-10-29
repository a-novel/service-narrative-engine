package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Beat is the REST representation of a beat.
type Beat struct {
	ID uuid.UUID `json:"id"`
	// ProjectID is the project the beat belongs to.
	ProjectID uuid.UUID `json:"projectID"`

	// Position orders the project's beats, from 0.
	Position int `json:"position"`

	// Title names the beat in the composer.
	Title string `json:"title"`
	// Color is the beat's color in the composer, as a lowercase #rrggbb hex code.
	Color string `json:"color"`
	// Description says what the beat is for, written for the author.
	Description string `json:"description"`
	// Prompt holds extra instructions for the model when it writes about the beat.
	Prompt string `json:"prompt"`

	UpdatedAt time.Time `json:"updatedAt"`
}

func loadBeat(beat *core.Beat) Beat {
	return Beat{
		ID:          beat.ID,
		ProjectID:   beat.ProjectID,
		Position:    beat.Position,
		Title:       beat.Title,
		Color:       beat.Color,
		Description: beat.Description,
		Prompt:      beat.Prompt,
		UpdatedAt:   beat.UpdatedAt,
	}
}
