package config

// ComposerPresetDefault holds the limits the service starts with. A cell holds a whole scene, so
// its limit leaves room for a long one.
var ComposerPresetDefault = Composer{
	MaxScenes:     ComposerMaxScenesDefault,
	MaxLayers:     ComposerMaxLayersDefault,
	MaxSeedScenes: ComposerMaxSeedScenesDefault,
	MaxCellBytes:  ComposerMaxCellBytesDefault,
}

const (
	// ComposerMaxScenesDefault is the default of [Composer.MaxScenes].
	ComposerMaxScenesDefault = 500
	// ComposerMaxLayersDefault is the default of [Composer.MaxLayers].
	ComposerMaxLayersDefault = 20
	// ComposerMaxSeedScenesDefault is the default of [Composer.MaxSeedScenes].
	ComposerMaxSeedScenesDefault = 100
	// ComposerMaxCellBytesDefault is the default of [Composer.MaxCellBytes], 256 KiB.
	ComposerMaxCellBytesDefault = 256 << 10
)
