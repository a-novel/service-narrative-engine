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

// RestLayerDeleteService is the core operation [RestLayerDelete] delegates to.
type RestLayerDeleteService interface {
	Exec(ctx context.Context, request *core.LayerDeleteRequest) ([]*core.Layer, error)
}

// RestLayerDeleteRequest is the query the delete-layers endpoint accepts.
type RestLayerDeleteRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
	// IDs lists the layers to delete, one id parameter each.
	IDs []uuid.UUID `schema:"id"`
}

// RestLayerDelete is the HTTP handler for the delete-layers endpoint. It responds with the deleted layers, as they
// were.
type RestLayerDelete struct {
	service RestLayerDeleteService
	logger  logging.Log
}

func NewRestLayerDelete(service RestLayerDeleteService, logger logging.Log) *RestLayerDelete {
	return &RestLayerDelete{service: service, logger: logger}
}

func (handler *RestLayerDelete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.LayerDelete")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestLayerDeleteRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.LayerDeleteRequest{
		OwnerID: ownerID, ProjectID: request.ProjectID, IDs: request.IDs,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(layer *core.Layer, _ int) Layer { return loadLayer(layer) })
	httpf.SendJSONStatus(w, span, http.StatusOK, response)
}
