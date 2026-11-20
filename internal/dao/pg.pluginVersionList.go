package dao

import (
	"context"
	_ "embed"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.pluginVersionList.sql
var pluginVersionListQuery string

// PluginVersionListRequest is the input to [PgPluginVersionList.Exec].
type PluginVersionListRequest struct {
	// After names the plugin whose releases to read, and the version they come after. The registry need not
	// hold that version.
	After PluginVersionKey

	// Limit caps the releases in the page.
	Limit int
	// Offset is how many releases come before the page, oldest first.
	Offset int
}

// PluginVersionListRow is one release of a page: a step of an upgrade.
type PluginVersionListRow struct {
	PluginVersionKey

	// HasMigration reports whether the release ships a migration module.
	HasMigration bool `bun:"has_migration"`
	// SchemaChecksum is the SHA-256 of the stored schema in lowercase hex. Two releases with the same one
	// share their schema.
	SchemaChecksum string `bun:"schema_checksum"`

	// Total counts every release after the version, beyond the page.
	Total int `bun:"total"`
}

// PgPluginVersionList reads a page of the releases of a plugin that come after a version, oldest first. Each
// row carries the total, and a page past the end comes back empty. A prerelease is never listed, and the
// release of a version comes after its prereleases.
type PgPluginVersionList struct{}

func NewPgPluginVersionList() *PgPluginVersionList {
	return &PgPluginVersionList{}
}

func (dao *PgPluginVersionList) Exec(
	ctx context.Context, request *PluginVersionListRequest,
) ([]*PluginVersionListRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgPluginVersionList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*PluginVersionListRow

	err = db.NewRaw(
		pluginVersionListQuery,
		request.After.Plugin, request.After.Major, request.After.Minor, request.After.Patch, request.After.Prerelease,
		request.Limit, request.Offset,
	).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("plugin_versions.count", len(entities)))

	return entities, nil
}
