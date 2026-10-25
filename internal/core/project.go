package core

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// Project is a narrative in progress, owned by one user. It produces one manuscript.
type Project struct {
	ID      uuid.UUID
	OwnerID uuid.UUID
	Title   string
	Premise string
	// Language is the language the project is written in, as a BCP 47 tag.
	Language string
	// Revision counts the writes to the project's own fields. A write names the revision it read.
	Revision  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

func newProject(entity *dao.Project) *Project {
	return &Project{
		ID:        entity.ID,
		OwnerID:   entity.OwnerID,
		Title:     entity.Title,
		Premise:   entity.Premise,
		Language:  entity.Language,
		Revision:  entity.Revision,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}
