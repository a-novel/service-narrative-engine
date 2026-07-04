// Package migrations holds the SQL schema migrations for the service, embedded
// so the migration runner can apply them without reading from disk at runtime.
package migrations

import (
	"embed"
)

// Migrations embeds every SQL migration file in this directory.
//
//go:embed *.sql
var Migrations embed.FS
