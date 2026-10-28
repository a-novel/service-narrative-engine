package handlers

import (
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// LayerPage is the REST representation of a page of a project's layers.
type LayerPage struct {
	Layers []Layer `json:"layers"`
	// Total counts every layer of the project, so a client paging through them knows when it holds
	// them all.
	Total int `json:"total"`
}

func loadLayerPage(page *core.LayerPage) LayerPage {
	return LayerPage{
		Layers: lo.Map(page.Layers, loadLayerMap),
		Total:  page.Total,
	}
}
