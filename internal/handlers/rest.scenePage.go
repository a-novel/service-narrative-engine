package handlers

import (
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// ScenePage is the REST representation of a page of a project's scenes.
type ScenePage struct {
	Scenes []Scene `json:"scenes"`
	// Total counts every scene of the project, so a client paging through them knows when it holds
	// them all.
	Total int `json:"total"`
}

func loadScenePage(page *core.ScenePage) ScenePage {
	return ScenePage{
		Scenes: lo.Map(page.Scenes, loadSceneMap),
		Total:  page.Total,
	}
}
