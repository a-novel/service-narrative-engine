package core

// ProjectComposer is a project with the layout of its Scenes Composer: its scenes and layers in
// order. A client reads the cells page by page, with [CellList].
type ProjectComposer struct {
	Project *Project
	Scenes  []*Scene
	Layers  []*Layer
}
