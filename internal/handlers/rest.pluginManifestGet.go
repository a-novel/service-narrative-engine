package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
)

// RestPluginManifestGetService is the core operation [RestPluginManifestGet] delegates to.
type RestPluginManifestGetService interface {
	Exec(ctx context.Context, request *core.PluginManifestGetRequest) (*core.PluginManifest, error)
}

// RestPluginManifestGetRequest is the query parameters the get-manifest endpoint accepts.
type RestPluginManifestGetRequest struct {
	Plugin  string `schema:"plugin"`
	Version string `schema:"version"`
}

// RestPluginManifestGet is the HTTP handler for the get-manifest endpoint. It responds with the manifest of
// one version of a plugin.
type RestPluginManifestGet struct {
	service RestPluginManifestGetService
	logger  logging.Log
}

func NewRestPluginManifestGet(service RestPluginManifestGetService, logger logging.Log) *RestPluginManifestGet {
	return &RestPluginManifestGet{service: service, logger: logger}
}

func (handler *RestPluginManifestGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.PluginManifestGet")
	defer span.End()

	var request RestPluginManifestGetRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.PluginManifestGetRequest{
		Plugin:  request.Plugin,
		Version: request.Version,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, pluginErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadPluginManifest(result))
}
