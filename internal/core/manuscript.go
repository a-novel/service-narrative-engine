package core

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrStepValueConflict reports content already saved for the same immutable step.
var ErrStepValueConflict = errors.New("step value already exists")

// ManuscriptFormat is the closed Stage 2 Manuscript format vocabulary.
type ManuscriptFormat string

const (
	// ManuscriptFormatProse is a scene sequence expressed as typed text blocks.
	ManuscriptFormatProse ManuscriptFormat = "prose"
)

// ManuscriptBlockKind identifies how one Manuscript block should be interpreted.
type ManuscriptBlockKind string

const (
	ManuscriptBlockKindProse    ManuscriptBlockKind = "prose"
	ManuscriptBlockKindDialogue ManuscriptBlockKind = "dialogue"
	ManuscriptBlockKindCue      ManuscriptBlockKind = "cue"
)

// ManuscriptBlock is one typed block inside a scene.
type ManuscriptBlock struct {
	Kind ManuscriptBlockKind `json:"kind" validate:"required,oneof=prose dialogue cue"`
	Text string              `json:"text" validate:"required,notblank"`
}

// ManuscriptScene is one ordered scene in a Manuscript proposal.
type ManuscriptScene struct {
	Title  string            `json:"title"  validate:"required,notblank"`
	Blocks []ManuscriptBlock `json:"blocks" validate:"required,min=1,dive"`
}

// ManuscriptValue is the typed, self-contained Stage 2 project document.
type ManuscriptValue struct {
	Title  string            `json:"title"  validate:"required,notblank"`
	Format ManuscriptFormat  `json:"format" validate:"required,oneof=prose"`
	Scenes []ManuscriptScene `json:"scenes" validate:"required,min=1,dive"`
}

// Manuscript is a persisted, self-contained project document.
type Manuscript struct {
	ID        uuid.UUID       `json:"id"`
	IdeaID    uuid.UUID       `json:"ideaID"`
	Value     ManuscriptValue `json:"value"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt *time.Time      `json:"updatedAt"`
}
