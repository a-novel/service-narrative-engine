package handlers

import (
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// ProjectComposer is the REST representation of a project with its whole Scenes Composer.
type ProjectComposer struct {
	Project

	Scenes []Scene `json:"scenes"`
	Layers []Layer `json:"layers"`
	Cells  []Cell  `json:"cells"`
}

func loadProjectComposer(composer *core.ProjectComposer) ProjectComposer {
	return ProjectComposer{
		Project: loadProject(composer.Project),
		Scenes:  lo.Map(composer.Scenes, loadSceneMap),
		Layers:  lo.Map(composer.Layers, loadLayerMap),
		Cells:   lo.Map(composer.Cells, loadCellMap),
	}
}
