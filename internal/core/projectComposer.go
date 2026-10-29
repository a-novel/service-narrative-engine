package core

// ProjectComposer is a new project with the layers it starts with, in order. Its beats and scenes come next,
// from [BeatsCreate] and [ScenesCreate].
type ProjectComposer struct {
	Project *Project
	// Layers are the project's required layers, one of each kind.
	Layers []*Layer
}
