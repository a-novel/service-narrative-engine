package core

// ProjectPage is a page of the caller's projects.
type ProjectPage struct {
	Projects []*Project
	// Total counts every project of the caller, so a client paging through them knows when it holds
	// them all. A page past the end reports 0.
	Total int
}
