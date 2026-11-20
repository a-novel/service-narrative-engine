package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestPluginListService is the core operation [RestPluginList] delegates to.
type RestPluginListService interface {
	Exec(ctx context.Context, request *core.PluginListRequest) (*core.PluginPage, error)
}

// RestPluginListRequest is the query parameters the list-plugins endpoint accepts.
type RestPluginListRequest struct {
	// Language is the language to read each title and description in. English when absent.
	Language string `schema:"language"`

	// Limit caps the page. Zero takes the default size.
	Limit int `schema:"limit"`
	// Offset is how many plugins come before the page, by name.
	Offset int `schema:"offset"`
}

// RestPluginList is the HTTP handler for the list-plugins endpoint. It responds with a page of the registry's
// plugins, by name, each with its latest release, and their total.
type RestPluginList struct {
	service RestPluginListService
	logger  logging.Log
}

func NewRestPluginList(service RestPluginListService, logger logging.Log) *RestPluginList {
	return &RestPluginList{service: service, logger: logger}
}

func (handler *RestPluginList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.PluginList")
	defer span.End()

	var request RestPluginListRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.PluginListRequest{
		Language: request.Language,
		Limit:    request.Limit,
		Offset:   request.Offset,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, pluginErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadPluginPage(result))
}
