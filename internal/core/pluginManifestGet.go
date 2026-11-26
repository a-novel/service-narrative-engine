package core

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// PluginManifestGetDao reads the manifest of one version of a plugin.
type PluginManifestGetDao interface {
	Exec(ctx context.Context, request *dao.PluginManifestSelectRequest) (*dao.PluginManifest, error)
}

// PluginManifestGetRequest names one version of a plugin.
type PluginManifestGetRequest struct {
	// Plugin is the plugin's name, a DNS label (RFC 1035).
	Plugin string `validate:"required,dns_rfc1035_label"`
	// Version is the version to read, as a semantic version without build metadata.
	Version string `validate:"required,pluginversion"`
}

// PluginManifestGet reads the manifest of one version of a plugin, a prerelease included.
type PluginManifestGet struct {
	dao PluginManifestGetDao
}

func NewPluginManifestGet(dao PluginManifestGetDao) *PluginManifestGet {
	return &PluginManifestGet{dao: dao}
}

func (service *PluginManifestGet) Exec(
	ctx context.Context, request *PluginManifestGetRequest,
) (*PluginManifest, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.PluginManifestGet")
	defer span.End()

	span.SetAttributes(
		attribute.String("plugin.name", request.Plugin),
		attribute.String("plugin.version", request.Version),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	key, err := parsePluginVersion(request.Version)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	key.Plugin = request.Plugin

	entity, err := service.dao.Exec(ctx, &dao.PluginManifestSelectRequest{PluginVersionKey: key})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select manifest: %w", err))
	}

	return &PluginManifest{
		Plugin:         entity.Plugin,
		Version:        formatPluginVersion(entity.PluginVersionKey),
		Schema:         entity.Schema,
		Exports:        entity.Exports,
		HasMigration:   entity.HasMigration,
		SchemaChecksum: entity.SchemaChecksum,
		UIChecksum:     entity.UIChecksum,
		PublishedAt:    entity.PublishedAt,
	}, nil
}
