package core

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// Manuscript is the text one manuscript layer holds across the scenes of its project.
type Manuscript struct {
	// Title names the narrative.
	Title string
	// Language is the language the project is written in, as a BCP 47 tag.
	Language string

	// Beats holds every beat of the project, in narrative order, those without scenes included.
	Beats []*ManuscriptBeat
	// Scenes holds every scene of the project, in narrative order.
	Scenes []*ManuscriptScene
}

// ManuscriptBeat is one beat of a [Manuscript].
type ManuscriptBeat struct {
	// Title names the beat.
	Title string
	// Description says what the beat is for, written for the author.
	Description string
}

// ManuscriptScene is one scene of a [Manuscript].
type ManuscriptScene struct {
	// Beat is the position of the scene's beat in [Manuscript.Beats].
	Beat int

	// Title names the scene.
	Title string

	// Blocks is the scene's text, a JSON array of blocks in reading order, empty when the layer holds no
	// text for the scene.
	Blocks json.RawMessage
}

// manuscriptContent reads the blocks of a manuscript cell, as they are stored.
type manuscriptContent struct {
	Blocks json.RawMessage `json:"blocks"`
}

// newManuscript builds the manuscript of a project from its rows in narrative order. A row whose beat
// differs from the previous row's opens the next beat.
func newManuscript(project *dao.Project, rows []*dao.ManuscriptRow) (*Manuscript, error) {
	manuscript := &Manuscript{
		Title:    project.Title,
		Language: project.Language,
		Beats:    []*ManuscriptBeat{},
		Scenes:   []*ManuscriptScene{},
	}

	var beatID uuid.UUID

	for _, row := range rows {
		if row.BeatID != beatID {
			beatID = row.BeatID
			manuscript.Beats = append(manuscript.Beats, &ManuscriptBeat{
				Title: row.BeatTitle, Description: row.BeatDescription,
			})
		}

		// A beat that holds no scene has a row of its own, with no scene.
		if row.SceneTitle == nil {
			continue
		}

		scene, err := newManuscriptScene(row, len(manuscript.Beats)-1)
		if err != nil {
			return nil, fmt.Errorf("scene %d: %w", len(manuscript.Scenes), err)
		}

		manuscript.Scenes = append(manuscript.Scenes, scene)
	}

	return manuscript, nil
}

func newManuscriptScene(row *dao.ManuscriptRow, beat int) (*ManuscriptScene, error) {
	// A scene the layer holds no cell for has no blocks.
	content := manuscriptContent{Blocks: json.RawMessage(`[]`)}

	if row.Content != nil {
		err := json.Unmarshal(row.Content, &content)
		if err != nil {
			return nil, fmt.Errorf("decode content: %w", err)
		}
	}

	return &ManuscriptScene{Beat: beat, Title: *row.SceneTitle, Blocks: content.Blocks}, nil
}
