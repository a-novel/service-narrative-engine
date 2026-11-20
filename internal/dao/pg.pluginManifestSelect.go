package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.pluginManifestSelect.sql
var pluginManifestSelectQuery string

// PluginManifest describes one version of a plugin: everything it publishes but its modules.
type PluginManifest struct {
	PluginVersionKey

	// Schema is the JSON Schema the content of an instance follows.
	Schema json.RawMessage `bun:"schema"`
	// Exports holds the lists of linkable items an instance exposes, by name.
	Exports json.RawMessage `bun:"exports"`
	// HasMigration reports whether the version ships a migration module.
	HasMigration bool `bun:"has_migration"`

	// SchemaChecksum is the SHA-256 of the stored schema in lowercase hex.
	SchemaChecksum string `bun:"schema_checksum"`
	// UIChecksum is the SHA-256 of the UI module in lowercase hex.
	UIChecksum string `bun:"ui_checksum"`

	PublishedAt time.Time `bun:"published_at"`
}

// PluginManifestSelectRequest is the input to [PgPluginManifestSelect.Exec].
type PluginManifestSelectRequest struct {
	PluginVersionKey
}

// PgPluginManifestSelect reads the manifest of one version of a plugin.
type PgPluginManifestSelect struct{}

func NewPgPluginManifestSelect() *PgPluginManifestSelect {
	return &PgPluginManifestSelect{}
}

func (dao *PgPluginManifestSelect) Exec(
	ctx context.Context, request *PluginManifestSelectRequest,
) (*PluginManifest, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgPluginManifestSelect")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entity := &PluginManifest{}

	err = db.NewRaw(
		pluginManifestSelectQuery,
		request.Plugin, request.Major, request.Minor, request.Patch, request.Prerelease,
	).Scan(ctx, entity)
	if errors.Is(err, sql.ErrNoRows) {
		err = errors.Join(err, ErrPluginVersionNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entity, nil
}
