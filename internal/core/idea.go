package core

import (
	"time"

	"github.com/google/uuid"
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
