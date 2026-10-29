package handlers

import (
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// ProjectComposer is the REST representation of a new project with the layers it starts with. Its
// scenes come next, from the create-scenes endpoint.
type ProjectComposer struct {
	Project

	// Layers are the project's required layers, one of each kind, in order.
	Layers []Layer `json:"layers"`
}

func loadProjectComposer(composer *core.ProjectComposer) ProjectComposer {
	return ProjectComposer{
		Project: loadProject(composer.Project),
		Layers:  lo.Map(composer.Layers, func(layer *core.Layer, _ int) Layer { return loadLayer(layer) }),
	}
}
