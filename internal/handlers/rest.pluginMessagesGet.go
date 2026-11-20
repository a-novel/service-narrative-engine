package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestPluginMessagesGetService is the core operation [RestPluginMessagesGet] delegates to.
type RestPluginMessagesGetService interface {
	Exec(ctx context.Context, request *core.PluginMessagesGetRequest) (*core.PluginMessages, error)
}

// RestPluginMessagesGetRequest is the query parameters the get-messages endpoint accepts.
type RestPluginMessagesGetRequest struct {
	Plugin  string `schema:"plugin"`
	Version string `schema:"version"`

	Language string `schema:"language"`
}

// RestPluginMessagesGet is the HTTP handler for the get-messages endpoint. It responds with the translations
// of one version of a plugin in one language.
type RestPluginMessagesGet struct {
	service RestPluginMessagesGetService
	logger  logging.Log
}

func NewRestPluginMessagesGet(service RestPluginMessagesGetService, logger logging.Log) *RestPluginMessagesGet {
	return &RestPluginMessagesGet{service: service, logger: logger}
}

func (handler *RestPluginMessagesGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.PluginMessagesGet")
	defer span.End()

	var request RestPluginMessagesGetRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.PluginMessagesGetRequest{
		Plugin:   request.Plugin,
		Version:  request.Version,
		Language: request.Language,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, pluginErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, PluginMessages{
		Language:    result.Language,
		Title:       result.Title,
		Description: result.Description,
		Messages:    result.Messages,
	})
}
