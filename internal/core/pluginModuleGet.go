package core

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// PluginModuleGetDao reads one module of a version of a plugin.
type PluginModuleGetDao interface {
	Exec(ctx context.Context, request *dao.PluginModuleSelectRequest) (*dao.PluginModule, error)
}

// PluginModuleGetRequest names one module of one version of a plugin.
type PluginModuleGetRequest struct {
	// Plugin is the plugin's name, a DNS label (RFC 1035).
	Plugin string `validate:"required,dns_rfc1035_label"`
	// Version is the version to read, as a semantic version without build metadata.
	Version string `validate:"required,pluginversion"`

	// Module names the module to read.
	Module PluginModuleKind `validate:"required,pluginmodule"`
}

// PluginModuleGet reads one JavaScript module of a version of a plugin, with its checksum. It returns
// [ErrPluginModuleNotFound] for a version the registry lacks, and for one that ships without the module.
type PluginModuleGet struct {
	dao PluginModuleGetDao
}

func NewPluginModuleGet(dao PluginModuleGetDao) *PluginModuleGet {
	return &PluginModuleGet{dao: dao}
}

func (service *PluginModuleGet) Exec(ctx context.Context, request *PluginModuleGetRequest) (*PluginModule, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.PluginModuleGet")
	defer span.End()

	span.SetAttributes(
		attribute.String("plugin.name", request.Plugin),
		attribute.String("plugin.version", request.Version),
		attribute.String("plugin.module", string(request.Module)),
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

	entity, err := service.dao.Exec(ctx, &dao.PluginModuleSelectRequest{
		PluginVersionKey: key,
		Module:           string(request.Module),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select module: %w", err))
	}

	return &PluginModule{Code: entity.Code, Checksum: entity.Checksum, PublishedAt: entity.PublishedAt}, nil
}
