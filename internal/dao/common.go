package dao

import "errors"

const (
	// pgForeignKeyViolation is the SQLSTATE PostgreSQL raises when a write breaks a foreign key.
	pgForeignKeyViolation = "23503"
	// pgCheckViolation is the SQLSTATE PostgreSQL raises when a write breaks a check constraint.
	pgCheckViolation = "23514"
)

var (
	// ErrPluginVersionNotFound is returned when the registry holds no such version of the plugin.
	ErrPluginVersionNotFound = errors.New("plugin version not found")
	// ErrPluginModuleNotFound is returned when the registry holds no such version of the plugin, or the
	// version ships without the module.
	ErrPluginModuleNotFound = errors.New("plugin module not found")
	// ErrPluginMessagesNotFound is returned when the registry holds no such version of the plugin, or the
	// version is not translated in the language.
	ErrPluginMessagesNotFound = errors.New("plugin messages not found")
)
