package handlers

import (
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// ProjectComposer is the REST representation of a project with the layout of its Scenes Composer:
// its scenes and layers in order. The cells come page by page from the list-cells endpoint.
type ProjectComposer struct {
	Project

	Scenes []Scene `json:"scenes"`
	Layers []Layer `json:"layers"`
}

func loadProjectComposer(composer *core.ProjectComposer) ProjectComposer {
	return ProjectComposer{
		Project: loadProject(composer.Project),
		Scenes:  lo.Map(composer.Scenes, loadSceneMap),
		Layers:  lo.Map(composer.Layers, loadLayerMap),
	}
}
