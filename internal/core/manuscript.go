package core

import (
	"encoding/json"
	"fmt"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// Manuscript is the text one manuscript layer holds across the scenes of its project.
type Manuscript struct {
	// Title names the narrative.
	Title string
	// Language is the language the project is written in, as a BCP 47 tag.
	Language string

	// Scenes holds every scene of the project, in narrative order.
	Scenes []*ManuscriptScene
}

// ManuscriptScene is one scene of a [Manuscript].
type ManuscriptScene struct {
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

func newManuscriptScene(entity *dao.ManuscriptScene) (*ManuscriptScene, error) {
	// A scene the layer holds no cell for has no blocks.
	content := manuscriptContent{Blocks: json.RawMessage(`[]`)}

	if entity.Content != nil {
		err := json.Unmarshal(entity.Content, &content)
		if err != nil {
			return nil, fmt.Errorf("decode content: %w", err)
		}
	}

	return &ManuscriptScene{Title: entity.Title, Blocks: content.Blocks}, nil
}
