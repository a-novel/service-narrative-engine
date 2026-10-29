package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestLayersDeleteService is the core operation [RestLayersDelete] delegates to.
type RestLayersDeleteService interface {
	Exec(ctx context.Context, request *core.LayersDeleteRequest) ([]*core.Layer, error)
}

// RestLayersDeleteRequest is the query the delete-layers endpoint accepts.
type RestLayersDeleteRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
	// IDs lists the layers to delete, one id parameter each.
	IDs []uuid.UUID `schema:"id"`
}

// RestLayersDelete is the HTTP handler for the delete-layers endpoint. It responds with the deleted layers, as they
// were.
type RestLayersDelete struct {
	service RestLayersDeleteService
	logger  logging.Log
}

func NewRestLayersDelete(service RestLayersDeleteService, logger logging.Log) *RestLayersDelete {
	return &RestLayersDelete{service: service, logger: logger}
}

func (handler *RestLayersDelete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.LayersDelete")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestLayersDeleteRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.LayersDeleteRequest{
		OwnerID: ownerID, ProjectID: request.ProjectID, IDs: request.IDs,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(layer *core.Layer, _ int) Layer { return loadLayer(layer) })
	httpf.SendJSONStatus(w, span, http.StatusOK, response)
}
