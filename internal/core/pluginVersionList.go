package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// PluginVersionListDao reads a page of the releases of a plugin that come after a version.
type PluginVersionListDao interface {
	Exec(ctx context.Context, request *dao.PluginVersionListRequest) ([]*dao.PluginVersionListRow, error)
}

// PluginVersionListLimitDefault is the page size of a [PluginVersionListRequest] whose Limit is zero.
const PluginVersionListLimitDefault = 100

// PluginVersionListRequest selects a page of the releases of a plugin that come after a version.
type PluginVersionListRequest struct {
	// Plugin is the plugin's name, a DNS label (RFC 1035).
	Plugin string `validate:"required,dns_rfc1035_label"`
	// After is the version the releases come after, as a semantic version without build metadata. The registry
	// need not hold it.
	After string `validate:"required,pluginversion"`

	// Limit caps the page at 500 releases. It defaults to [PluginVersionListLimitDefault] when zero.
	Limit int `validate:"min=0,max=500"`
	// Offset is how many releases come before the page, oldest first.
	Offset int `validate:"min=0"`
}

// PluginVersionList reads the releases of a plugin that come after a version, page by page and oldest first:
// the steps of an upgrade from that version. Each page carries the total, so a client knows when it holds
// every step. A prerelease is never a step, and the release of a version comes after its prereleases. A plugin
// the registry lacks has no release, so its page is empty.
type PluginVersionList struct {
	dao PluginVersionListDao
}

func NewPluginVersionList(dao PluginVersionListDao) *PluginVersionList {
	return &PluginVersionList{dao: dao}
}

func (service *PluginVersionList) Exec(
	ctx context.Context, request *PluginVersionListRequest,
) (*PluginVersionPage, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.PluginVersionList")
	defer span.End()

	span.SetAttributes(
		attribute.String("plugin.name", request.Plugin),
		attribute.String("plugin_versions.after", request.After),
		attribute.Int("plugin_versions.limit", request.Limit),
		attribute.Int("plugin_versions.offset", request.Offset),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	after, err := parsePluginVersion(request.After)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	after.Plugin = request.Plugin

	if request.Limit == 0 {
		request.Limit = PluginVersionListLimitDefault
	}

	rows, err := service.dao.Exec(ctx, &dao.PluginVersionListRequest{
		After:  after,
		Limit:  request.Limit,
		Offset: request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list plugin versions: %w", err))
	}

	page := &PluginVersionPage{
		Versions: lo.Map(rows, func(row *dao.PluginVersionListRow, _ int) *PluginVersion {
			return &PluginVersion{
				Version:        formatPluginVersion(row.PluginVersionKey),
				HasMigration:   row.HasMigration,
				SchemaChecksum: row.SchemaChecksum,
			}
		}),
	}

	// Every row carries the total, and a page past the end carries none.
	if len(rows) > 0 {
		page.Total = rows[0].Total
	}

	return page, nil
}
