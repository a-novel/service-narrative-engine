package core

// ProjectComposer is a project with its whole Scenes Composer: its scenes and layers in order, and
// their cells.
type ProjectComposer struct {
	Project *Project
	Scenes  []*Scene
	Layers  []*Layer
	Cells   []*Cell
}
