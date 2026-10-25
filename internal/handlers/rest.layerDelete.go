package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestLayerDeleteService is the core operation [RestLayerDelete] delegates to.
type RestLayerDeleteService interface {
	Exec(ctx context.Context, request *core.LayerDeleteRequest) (*core.Layer, error)
}

// RestLayerDeleteRequest is the query parameters the delete-layer endpoint accepts.
type RestLayerDeleteRequest struct {
	ID       uuid.UUID `schema:"id"`
	Revision int       `schema:"revision"`
}

// RestLayerDelete is the HTTP handler for the delete-layer endpoint. It responds with the deleted
// layer.
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

	serviceRequest := &core.LayerDeleteRequest{
		OwnerID:  ownerID,
		ID:       request.ID,
		Revision: request.Revision,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadLayer(result))
}
