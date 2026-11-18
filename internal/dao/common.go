package dao

const (
	// pgForeignKeyViolation is the SQLSTATE PostgreSQL raises when a write breaks a foreign key.
	pgForeignKeyViolation = "23503"
	// pgCheckViolation is the SQLSTATE PostgreSQL raises when a write breaks a check constraint.
	pgCheckViolation = "23514"
)
