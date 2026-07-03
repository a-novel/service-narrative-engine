package configtest

import (
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-narrative-engine/internal/config/env"
)

// PostgresPreset is the PostgreSQL configuration used by integration tests
// (DAO tests and health/status handler tests). It points at the same database
// as the production preset; the test harness (postgres.RunTransactionalTest)
// isolates each test in a rolled-back transaction.
//
// The Go toolchain has no notion of a "test-only" package (only the `_test.go`
// file suffix is excluded from production builds, and that suffix wouldn't work
// here because other test packages need to import this preset). The boundary is
// a convention: production code never imports `configtest`, and a stray import
// would be caught in review. See the write-go-tests skill, "Cross-Package Test
// Fixtures".
var PostgresPreset = postgrespresets.NewDefault(pgdriver.WithDSN(env.PostgresDsn))
