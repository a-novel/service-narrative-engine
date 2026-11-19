package handlers

import (
	"bytes"
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestPluginModuleGetService is the core operation [RestPluginModuleGet] delegates to.
type RestPluginModuleGetService interface {
	Exec(ctx context.Context, request *core.PluginModuleGetRequest) (*core.PluginModule, error)
}

// RestPluginModuleGetRequest is the query parameters a get-module endpoint accepts.
type RestPluginModuleGetRequest struct {
	Plugin  string `schema:"plugin"`
	Version string `schema:"version"`
}

// RestPluginModuleGet is the HTTP handler for the get-module endpoints, one per module of
// [core.PluginModules]. It responds with the source of that module for one version of a plugin.
//
// The module is JavaScript, which X-Content-Type-Options keeps the browser from second-guessing. Its
// checksum is its strong ETag (RFC 9110 section 8.8.3), and Last-Modified is when its version was published.
// [http.ServeContent] then answers a matching conditional request with 304. A published version never
// changes, so Cache-Control marks the module immutable (RFC 8246): a cache keeps it for a year without
// asking again. The response answers a signed-in user, so only that user's own cache stores it.
type RestPluginModuleGet struct {
	service RestPluginModuleGetService
	// module names the module the endpoint serves.
	module string
	logger logging.Log
}

func NewRestPluginModuleGet(
	service RestPluginModuleGetService, module string, logger logging.Log,
) *RestPluginModuleGet {
	return &RestPluginModuleGet{service: service, module: module, logger: logger}
}

func (handler *RestPluginModuleGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.PluginModuleGet")
	defer span.End()

	var request RestPluginModuleGetRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.PluginModuleGetRequest{
		Plugin:  request.Plugin,
		Version: request.Version,
		Module:  handler.module,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, pluginErrors, err)

		return
	}

	header := w.Header()
	header.Set("Content-Type", "text/javascript; charset=utf-8")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Cache-Control", "private, max-age=31536000, immutable")
	header.Set("ETag", `"`+result.Checksum+`"`)

	http.ServeContent(w, r, "", result.PublishedAt, bytes.NewReader(result.Code))
}
