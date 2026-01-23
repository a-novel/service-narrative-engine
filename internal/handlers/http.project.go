package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/services"
)

type Project struct {
	ID        uuid.UUID `json:"id"`
	Owner     uuid.UUID `json:"owner"`
	Lang      string    `json:"lang"`
	Title     string    `json:"title"`
	Workflow  []string  `json:"workflow"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func loadProject(s *services.Project) Project {
	return Project{
		ID:        s.ID,
		Owner:     s.Owner,
		Lang:      s.Lang,
		Title:     s.Title,
		Workflow:  s.Workflow,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func loadProjectMap(p *services.Project, _ int) Project {
	return loadProject(p)
}
