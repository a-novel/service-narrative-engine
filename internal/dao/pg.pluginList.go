package dao

import (
	"context"
	_ "embed"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.pluginList.sql
var pluginListQuery string

// PluginListRequest is the input to [PgPluginList.Exec].
type PluginListRequest struct {
	// Language is the language to read each title and description in, as a BCP 47 tag. A plugin that is not
	// translated in it reads in English.
	Language string

	// Limit caps the plugins in the page.
	Limit int
	// Offset is how many plugins come before the page, by name.
	Offset int
}

// PluginListRow is one plugin of a page, as its latest release describes it.
type PluginListRow struct {
	// PluginVersionKey names the plugin and its latest release. A prerelease is never one.
	PluginVersionKey

	// Title names the plugin.
	Title string `bun:"title"`
	// Description says what the plugin does.
	Description string `bun:"description"`

	// Total counts every plugin that has a release, beyond the page.
	Total int `bun:"total"`
}

// PgPluginList reads a page of the registry's plugins, by name. Each row carries the plugin's latest release
// and the total, and a page past the end comes back empty. A plugin that only has prereleases is left out.
type PgPluginList struct{}

func NewPgPluginList() *PgPluginList {
	return &PgPluginList{}
}

func (dao *PgPluginList) Exec(ctx context.Context, request *PluginListRequest) ([]*PluginListRow, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgPluginList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	var entities []*PluginListRow

	err = db.NewRaw(pluginListQuery, request.Language, request.Limit, request.Offset).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	span.SetAttributes(attribute.Int("plugins.count", len(entities)))

	return entities, nil
}
