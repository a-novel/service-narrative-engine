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

// PluginListDao reads a page of the registry's plugins.
type PluginListDao interface {
	Exec(ctx context.Context, request *dao.PluginListRequest) ([]*dao.PluginListRow, error)
}

// PluginListLimitDefault is the page size of a [PluginListRequest] whose Limit is zero.
const PluginListLimitDefault = 100

// PluginListRequest selects a page of the registry's plugins.
type PluginListRequest struct {
	// Language is the language to read each title and description in, as a BCP 47 tag. A plugin that is not
	// translated in it reads in English, and so does every plugin when it is empty.
	Language string `validate:"omitempty,bcp47_language_tag"`

	// Limit caps the page at 500 plugins. It defaults to [PluginListLimitDefault] when zero.
	Limit int `validate:"min=0,max=500"`
	// Offset is how many plugins come before the page, by name.
	Offset int `validate:"min=0"`
}

// PluginList reads the registry's plugins page by page, by name. Each plugin carries its latest release, and
// each page the total, so a client knows when it holds every plugin. A plugin that only has prereleases is
// left out.
type PluginList struct {
	dao PluginListDao
}

func NewPluginList(dao PluginListDao) *PluginList {
	return &PluginList{dao: dao}
}

func (service *PluginList) Exec(ctx context.Context, request *PluginListRequest) (*PluginPage, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.PluginList")
	defer span.End()

	span.SetAttributes(
		attribute.String("plugins.language", request.Language),
		attribute.Int("plugins.limit", request.Limit),
		attribute.Int("plugins.offset", request.Offset),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	if request.Limit == 0 {
		request.Limit = PluginListLimitDefault
	}

	rows, err := service.dao.Exec(ctx, &dao.PluginListRequest{
		Language: request.Language,
		Limit:    request.Limit,
		Offset:   request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list plugins: %w", err))
	}

	page := &PluginPage{
		Plugins: lo.Map(rows, func(row *dao.PluginListRow, _ int) *Plugin {
			return &Plugin{
				Name:        row.Plugin,
				Version:     formatPluginVersion(row.PluginVersionKey),
				Title:       row.Title,
				Description: row.Description,
			}
		}),
	}

	// Every row carries the total, and a page past the end carries none.
	if len(rows) > 0 {
		page.Total = rows[0].Total
	}

	return page, nil
}
