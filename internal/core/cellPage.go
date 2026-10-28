package core

// CellPage is a page of a project's cells.
type CellPage struct {
	Cells []*Cell

	// Total counts every cell of the project, so a client paging through them knows when it holds
	// them all. A page past the end reports 0.
	Total int
}
