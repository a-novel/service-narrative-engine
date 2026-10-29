package core

// BeatPage is a page of a project's beats.
type BeatPage struct {
	Beats []*Beat

	// Total counts every beat of the project, so a client paging through them knows when it holds
	// them all. A page past the end reports 0.
	Total int
}
