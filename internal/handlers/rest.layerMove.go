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

// RestLayerMoveService is the core operation [RestLayerMove] delegates to.
type RestLayerMoveService interface {
	Exec(ctx context.Context, request *core.LayerMoveRequest) (*core.Layer, error)
}

// RestLayerMoveRequest is the JSON body the move-layer endpoint accepts.
type RestLayerMoveRequest struct {
	ID    uuid.UUID `json:"id"`
	After uuid.UUID `json:"after"`
}

// RestLayerMove is the HTTP handler for the move-layer endpoint. It responds with the layer at its new
// position.
type RestLayerMove struct {
	service RestLayerMoveService
	logger  logging.Log
}

func NewRestLayerMove(service RestLayerMoveService, logger logging.Log) *RestLayerMove {
	return &RestLayerMove{service: service, logger: logger}
}

func (handler *RestLayerMove) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.LayerMove")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestLayerMoveRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.LayerMoveRequest{
		OwnerID: ownerID,
		ID:      request.ID,
		After:   request.After,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadLayer(result))
}
