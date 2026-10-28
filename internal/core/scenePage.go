package core

// ScenePage is a page of a project's scenes.
type ScenePage struct {
	Scenes []*Scene

	// Total counts every scene of the project, so a client paging through them knows when it holds
	// them all. A page past the end reports 0.
	Total int
}
