package handlers

import (
	"encoding/json"

	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

const (
	// manuscriptFormat names an exported manuscript in its format field.
	manuscriptFormat = "manuscript"
	// manuscriptFormatVersion is the version of the manuscript format the service exports.
	manuscriptFormatVersion = 1
)

// Manuscript is the REST representation of an exported manuscript, format v1. The schema
// internal/models/schemas/manuscript.schema.json defines it.
type Manuscript struct {
	// Format is always manuscript.
	Format string `json:"format"`
	// FormatVersion is the version of the format the document follows.
	FormatVersion int `json:"formatVersion"`

	// Title names the narrative.
	Title string `json:"title"`
	// Language is the language of the text, one of [core.ProjectLanguages].
	Language string `json:"language"`

	// Scenes holds every scene of the narrative, in narrative order.
	Scenes []ManuscriptScene `json:"scenes"`
	// Appendices holds data about the text, keyed by appendix name.
	Appendices ManuscriptAppendices `json:"appendices"`
}

// ManuscriptAppendices is the REST representation of the appendices of a [Manuscript].
type ManuscriptAppendices struct {
	// Beats holds every beat of the narrative, in narrative order. A scene names its beat by its position
	// here.
	Beats []ManuscriptBeat `json:"beats"`
}

// ManuscriptBeat is the REST representation of one beat of a [Manuscript].
type ManuscriptBeat struct {
	// Title names the beat.
	Title string `json:"title"`
	// Description says what the beat is for, written for the author.
	Description string `json:"description"`
}

// ManuscriptScene is the REST representation of one scene of a [Manuscript].
type ManuscriptScene struct {
	// Beat is the position of the scene's beat in the beats appendix.
	Beat int `json:"beat"`
	// Title names the scene.
	Title string `json:"title"`

	// Blocks is the scene's text, in reading order.
	Blocks json.RawMessage `json:"blocks"`
}

func loadManuscript(manuscript *core.Manuscript) Manuscript {
	return Manuscript{
		Format:        manuscriptFormat,
		FormatVersion: manuscriptFormatVersion,
		Title:         manuscript.Title,
		Language:      manuscript.Language,
		Scenes: lo.Map(manuscript.Scenes, func(scene *core.ManuscriptScene, _ int) ManuscriptScene {
			return ManuscriptScene{Beat: scene.Beat, Title: scene.Title, Blocks: scene.Blocks}
		}),
		Appendices: ManuscriptAppendices{
			Beats: lo.Map(manuscript.Beats, func(beat *core.ManuscriptBeat, _ int) ManuscriptBeat {
				return ManuscriptBeat{Title: beat.Title, Description: beat.Description}
			}),
		},
	}
}
