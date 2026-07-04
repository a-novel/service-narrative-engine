package config

import (
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-narrative-engine/internal/config/env"
)

// PostgresPresetDefault is the default PostgreSQL connection configuration, built from the
// POSTGRES_DSN environment variable.
var PostgresPresetDefault = postgrespresets.NewDefault(pgdriver.WithDSN(env.PostgresDsn))
