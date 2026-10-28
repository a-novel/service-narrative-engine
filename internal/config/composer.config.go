package config

// Composer bounds what one request writes to a project's Scenes Composer. The schema caps a project
// at 500 scenes and 20 layers.
type Composer struct {
	// MaxSeedScenes caps the scenes a project starts with.
	MaxSeedScenes int `json:"maxSeedScenes" yaml:"maxSeedScenes"`
	// MaxCellBytes caps the size of one cell's content, encoded as JSON.
	MaxCellBytes int `json:"maxCellBytes" yaml:"maxCellBytes"`
}
