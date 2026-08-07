package core

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// Idea is one immutable typed content version under a stable Project.
type Idea struct {
	ProjectID        uuid.UUID `json:"projectID"`
	VersionID        uuid.UUID `json:"versionID"`
	OwnerID          uuid.UUID `json:"ownerID"`
	Seed             string    `json:"seed"`
	Genre            string    `json:"genre"`
	Title            string    `json:"title"`
	ProjectCreatedAt time.Time `json:"projectCreatedAt"`
	CreatedAt        time.Time `json:"createdAt"`
}

// ideaFromDao maps stored Idea content without exposing database types to handlers.
func ideaFromDao(entity *dao.Idea) *Idea {
	if entity == nil {
		return nil
	}

	return &Idea{
		ProjectID:        entity.ProjectID,
		VersionID:        entity.VersionID,
		OwnerID:          entity.OwnerID,
		Seed:             entity.Seed,
		Genre:            entity.Genre,
		Title:            entity.Title,
		ProjectCreatedAt: entity.ProjectCreatedAt,
		CreatedAt:        entity.CreatedAt,
	}
}
