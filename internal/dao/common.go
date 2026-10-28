package dao

import (
	"errors"

	"github.com/uptrace/bun/driver/pgdriver"
)

// pgCheckViolation is the SQLSTATE PostgreSQL raises when a write breaks a check constraint.
const pgCheckViolation = "23514"

// isCheckViolation reports whether err comes from a write that broke the named check constraint.
func isCheckViolation(err error, constraint string) bool {
	var pgErr pgdriver.Error

	return errors.As(err, &pgErr) && pgErr.Field('C') == pgCheckViolation && pgErr.Field('n') == constraint
}
