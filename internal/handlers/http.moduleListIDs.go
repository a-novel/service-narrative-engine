package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/services"
)

type ModuleListIDsService interface {
	Exec(ctx context.Context, request *services.ModuleListIDsRequest) ([]string, error)
}

type ModuleListIDsRequest struct {
	Namespace string `schema:"namespace"`
	Limit     int    `schema:"limit"`
	Offset    int    `schema:"offset"`
}

type ModuleListIDs struct {
	service ModuleListIDsService
	logger  logging.Log
}

func NewModuleListIDs(service ModuleListIDsService, logger logging.Log) *ModuleListIDs {
	return &ModuleListIDs{service: service, logger: logger}
}

func (handler *ModuleListIDs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ModuleListIDs")
	defer span.End()

	var request ModuleListIDsRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.ModuleListIDsRequest{
		Namespace: request.Namespace,
		Limit:     request.Limit,
		Offset:    request.Offset,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			services.ErrInvalidRequest: http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, res)
}
