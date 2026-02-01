package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/services"
)

type ModuleListNamespacesService interface {
	Exec(ctx context.Context, request *services.ModuleListNamespacesRequest) ([]string, error)
}

type ModuleListNamespacesRequest struct {
	Limit  int `schema:"limit"`
	Offset int `schema:"offset"`
}

type ModuleListNamespaces struct {
	service ModuleListNamespacesService
	logger  logging.Log
}

func NewModuleListNamespaces(service ModuleListNamespacesService, logger logging.Log) *ModuleListNamespaces {
	return &ModuleListNamespaces{service: service, logger: logger}
}

func (handler *ModuleListNamespaces) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ModuleListNamespaces")
	defer span.End()

	var request ModuleListNamespacesRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.ModuleListNamespacesRequest{
		Limit:  request.Limit,
		Offset: request.Offset,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			services.ErrInvalidRequest: http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, res)
}
