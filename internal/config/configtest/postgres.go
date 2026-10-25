package configtest

import (
	"github.com/a-novel/service-template/internal/config"
)

// PostgresPreset is the PostgreSQL configuration the integration tests connect with. It aliases the
// production preset, so tests read the same discrete settings or DSN.
var PostgresPreset = config.PostgresPresetDefault
