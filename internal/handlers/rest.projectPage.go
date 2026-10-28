package handlers

import (
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// ProjectPage is the REST representation of a page of the caller's projects.
type ProjectPage struct {
	Projects []Project `json:"projects"`

	// Total counts every project of the caller, so a client paging through them knows when it holds them
	// all.
	Total int `json:"total"`
}

func loadProjectPage(page *core.ProjectPage) ProjectPage {
	return ProjectPage{
		Projects: lo.Map(page.Projects, func(project *core.Project, _ int) Project { return loadProject(project) }),
		Total:    page.Total,
	}
}
