package dao

import (
	"errors"

	"github.com/uptrace/bun/driver/pgdriver"
)

const (
	// pgForeignKeyViolation is the SQLSTATE PostgreSQL raises when a write breaks a foreign key.
	pgForeignKeyViolation = "23503"
	// pgCheckViolation is the SQLSTATE PostgreSQL raises when a write breaks a check constraint.
	pgCheckViolation = "23514"
)

// isViolation reports whether err comes from a write that broke the named constraint, which code
// classifies.
func isViolation(err error, code, constraint string) bool {
	pgErr, ok := errors.AsType[pgdriver.Error](err)

	return ok && pgErr.Field('C') == code && pgErr.Field('n') == constraint
}
