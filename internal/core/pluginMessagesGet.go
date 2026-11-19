package core

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// PluginMessagesGetDao reads the translations of one version of a plugin in one language.
type PluginMessagesGetDao interface {
	Exec(ctx context.Context, request *dao.PluginMessagesSelectRequest) (*dao.PluginMessages, error)
}

// PluginMessagesGetRequest names one language of one version of a plugin.
type PluginMessagesGetRequest struct {
	// Plugin is the plugin's name.
	Plugin string `validate:"required"`
	// Version is the version to read, as a semantic version without build metadata.
	Version string `validate:"required,pluginversion"`

	// Language is the language to read, as a BCP 47 tag.
	Language string `validate:"required,bcp47_language_tag"`
}

// PluginMessagesGet reads the translations of one version of a plugin in one language. It reads that language
// alone: a version that lacks it returns [ErrPluginMessagesNotFound], and the caller asks for English, which
// every version holds.
type PluginMessagesGet struct {
	dao PluginMessagesGetDao
}

func NewPluginMessagesGet(dao PluginMessagesGetDao) *PluginMessagesGet {
	return &PluginMessagesGet{dao: dao}
}

func (service *PluginMessagesGet) Exec(
	ctx context.Context, request *PluginMessagesGetRequest,
) (*PluginMessages, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.PluginMessagesGet")
	defer span.End()

	span.SetAttributes(
		attribute.String("plugin.name", request.Plugin),
		attribute.String("plugin.version", request.Version),
		attribute.String("plugin.language", request.Language),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.dao.Exec(ctx, &dao.PluginMessagesSelectRequest{
		PluginVersionKey: pluginVersionKey(request.Plugin, request.Version),
		Language:         request.Language,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select messages: %w", err))
	}

	return &PluginMessages{
		Language:    entity.Language,
		Title:       entity.Title,
		Description: entity.Description,
		Messages:    entity.Messages,
	}, nil
}
