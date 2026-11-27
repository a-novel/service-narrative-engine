package dao

import "errors"

const (
	// pgForeignKeyViolation is the SQLSTATE PostgreSQL raises when a write names a row its foreign key
	// does not find.
	pgForeignKeyViolation = "23503"
	// pgRestrictViolation is the SQLSTATE PostgreSQL raises when a write removes a row that a foreign
	// key with a restriction still names.
	pgRestrictViolation = "23001"
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
	// ErrInstanceNotFound is returned when the project holds no instance with the ID.
	ErrInstanceNotFound = errors.New("instance not found")
)
