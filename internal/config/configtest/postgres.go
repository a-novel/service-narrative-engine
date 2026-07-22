// Package configtest holds configuration presets shared across the service's
// integration tests.
package configtest

import (
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-narrative-engine/internal/config/env"
)

// PostgresPreset carries the connection options integration tests reach PostgreSQL with. The
// harness at [github.com/a-novel-kit/golib/postgres.RunDBTest] migrates one template database from
// them, then clones a throwaway database per test and drops it afterwards, so tests never observe
// each other's writes.
//
// It lives in a regular (non-_test.go) file so other packages' tests can import it: Go excludes
// _test.go files from a package's exported surface. Keeping production code out of configtest is a
// convention enforced in review.
var PostgresPreset = postgrespresets.NewDefault(pgdriver.WithDSN(env.PostgresDsn))
