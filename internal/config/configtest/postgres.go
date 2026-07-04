package configtest

import (
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-narrative-engine/internal/config/env"
)

// PostgresPreset is the PostgreSQL configuration used by integration tests. It targets the same
// database as the production preset; the transactional test harness isolates each test in a
// rolled-back transaction.
//
// Go has no test-only package: the `_test.go` suffix is the only build-time test exclusion, and it
// cannot apply here because other test packages import this preset. The boundary is a convention —
// production code never imports configtest.
var PostgresPreset = postgrespresets.NewDefault(pgdriver.WithDSN(env.PostgresDsn))
