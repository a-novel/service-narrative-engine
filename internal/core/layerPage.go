package core

// LayerPage is a page of a project's layers.
type LayerPage struct {
	Layers []*Layer

	// Total counts every layer of the project, so a client paging through them knows when it holds
	// them all. A page past the end reports 0.
	Total int
}
