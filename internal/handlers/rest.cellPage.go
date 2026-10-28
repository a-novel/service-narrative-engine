package handlers

import (
	"github.com/samber/lo"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// CellPage is the REST representation of a page of a project's cells.
type CellPage struct {
	Cells []Cell `json:"cells"`
	// Total counts every cell of the project, so a client paging through them knows when it holds them
	// all.
	Total int `json:"total"`
}

func loadCellPage(page *core.CellPage) CellPage {
	return CellPage{
		Cells: lo.Map(page.Cells, loadCellMap),
		Total: page.Total,
	}
}
