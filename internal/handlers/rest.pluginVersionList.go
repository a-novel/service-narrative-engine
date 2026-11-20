package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestPluginVersionListService is the core operation [RestPluginVersionList] delegates to.
type RestPluginVersionListService interface {
	Exec(ctx context.Context, request *core.PluginVersionListRequest) (*core.PluginVersionPage, error)
}

// RestPluginVersionListRequest is the query parameters the list-versions endpoint accepts.
type RestPluginVersionListRequest struct {
	Plugin string `schema:"plugin"`
	// After is the version the releases come after.
	After string `schema:"after"`

	// Limit caps the page. Zero takes the default size.
	Limit int `schema:"limit"`
	// Offset is how many releases come before the page, oldest first.
	Offset int `schema:"offset"`
}

// RestPluginVersionList is the HTTP handler for the list-versions endpoint. It responds with a page of the
// releases of a plugin that come after a version, oldest first, and their total.
type RestPluginVersionList struct {
	service RestPluginVersionListService
	logger  logging.Log
}

func NewRestPluginVersionList(service RestPluginVersionListService, logger logging.Log) *RestPluginVersionList {
	return &RestPluginVersionList{service: service, logger: logger}
}

func (handler *RestPluginVersionList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.PluginVersionList")
	defer span.End()

	var request RestPluginVersionListRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.PluginVersionListRequest{
		Plugin: request.Plugin,
		After:  request.After,
		Limit:  request.Limit,
		Offset: request.Offset,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, pluginErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadPluginVersionPage(result))
}
