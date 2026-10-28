package core

// ProjectComposer is a new project with the layers it starts with, in order. Its scenes come next, from
// [SceneCreate].
type ProjectComposer struct {
	Project *Project
	// Layers are the project's required layers, one of each kind.
	Layers []*Layer
}
