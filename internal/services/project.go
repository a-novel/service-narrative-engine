package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	ErrUserDoesNotOwnProject = errors.New("user is not the owner of this project")
	ErrModuleNotInProject    = errors.New("module is not in the project")
)

type Project struct {
	ID        uuid.UUID
	Owner     uuid.UUID
	Lang      string
	Title     string
	Workflow  []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func loadProject(project *dao.Project) *Project {
	return &Project{
		ID:        project.ID,
		Owner:     project.Owner,
		Lang:      project.Lang,
		Title:     project.Title,
		Workflow:  project.Workflow,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
	}
}

func loadProjectsMap(project *dao.Project, _ int) *Project {
	return loadProject(project)
}

// VerifyProjectOwnership assess that the given user has the proper access authorizations to edit the project.
func VerifyProjectOwnership(project *dao.Project, userID uuid.UUID) error {
	if project.Owner != userID {
		return ErrUserDoesNotOwnProject
	}

	return nil
}

// VerifyModule assess that the given module is part of the project's workflow.
// The module parameter should be a full versioned module string (e.g., "namespace:module@v1.0.0").
func VerifyModule(project *dao.Project, module string) error {
	for _, m := range project.Workflow {
		if m == module {
			return nil
		}
	}

	return fmt.Errorf("module '%s': %w", module, ErrModuleNotInProject)
}
