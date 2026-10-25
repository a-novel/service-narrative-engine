package config

// Composer bounds the size of a project's Scenes Composer.
type Composer struct {
	// MaxScenes caps the scenes of one project.
	MaxScenes int `json:"maxScenes" yaml:"maxScenes"`
	// MaxLayers caps the layers of one project, the manuscript layer included.
	MaxLayers int `json:"maxLayers" yaml:"maxLayers"`
	// MaxSeedScenes caps the scenes a project starts with.
	MaxSeedScenes int `json:"maxSeedScenes" yaml:"maxSeedScenes"`
	// MaxCellBytes caps the size of one cell's content, encoded as JSON.
	MaxCellBytes int `json:"maxCellBytes" yaml:"maxCellBytes"`
}
