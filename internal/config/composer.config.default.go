package config

// ComposerPresetDefault holds the limits the service starts with. A cell holds a whole scene, so
// its limit leaves room for a long one.
var ComposerPresetDefault = Composer{
	MaxSeedScenes: ComposerMaxSeedScenesDefault,
	MaxCellBytes:  ComposerMaxCellBytesDefault,
}

const (
	// ComposerMaxSeedScenesDefault is the default of [Composer.MaxSeedScenes].
	ComposerMaxSeedScenesDefault = 100
	// ComposerMaxCellBytesDefault is the default of [Composer.MaxCellBytes], 2 MiB.
	ComposerMaxCellBytesDefault = 2 << 20
)
