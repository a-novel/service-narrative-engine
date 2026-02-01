package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/services"
)

type ModuleListVersionsService interface {
	Exec(ctx context.Context, request *services.ModuleListVersionsRequest) ([]*services.ModuleVersion, error)
}

type ModuleListVersionsRequest struct {
	ID         string `schema:"id"`
	Namespace  string `schema:"namespace"`
	Limit      int    `schema:"limit"`
	Offset     int    `schema:"offset"`
	Version    string `schema:"version"`
	Preversion bool   `schema:"preversion"`
}

type ModuleListVersions struct {
	service ModuleListVersionsService
	logger  logging.Log
}

func NewModuleListVersions(service ModuleListVersionsService, logger logging.Log) *ModuleListVersions {
	return &ModuleListVersions{service: service, logger: logger}
}

func (handler *ModuleListVersions) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ModuleListVersions")
	defer span.End()

	var request ModuleListVersionsRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.ModuleListVersionsRequest{
		ID:         request.ID,
		Namespace:  request.Namespace,
		Limit:      request.Limit,
		Offset:     request.Offset,
		Version:    request.Version,
		Preversion: request.Preversion,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			services.ErrInvalidRequest: http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, lo.Map(res, loadModuleVersionsMap))
}
