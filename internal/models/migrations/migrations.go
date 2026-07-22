// Package migrations holds the service's database schema as an ordered set of SQL migration files,
// embedded into the binary so the schema ships with the service.
package migrations

import (
	"embed"
)

// Migrations is the embedded filesystem of SQL migration files. Pass it to the migration runner to
// bring a database up to the service's current schema. Each timestamped file has an .up.sql step to
// apply the change and a matching .down.sql step to revert it.
//
//go:embed *.sql
var Migrations embed.FS
