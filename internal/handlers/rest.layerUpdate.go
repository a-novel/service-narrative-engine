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

// RestLayerUpdateService is the core operation [RestLayerUpdate] delegates to.
type RestLayerUpdateService interface {
	Exec(ctx context.Context, request *core.LayerUpdateRequest) (*core.Layer, error)
}

// RestLayerUpdateRequest is the JSON body the update-layer endpoint accepts.
type RestLayerUpdateRequest struct {
	ID          uuid.UUID `json:"id"`
	Revision    int       `json:"revision"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// RestLayerUpdate is the HTTP handler for the update-layer endpoint. It responds with the updated
// layer.
type RestLayerUpdate struct {
	service RestLayerUpdateService
	logger  logging.Log
}

func NewRestLayerUpdate(service RestLayerUpdateService, logger logging.Log) *RestLayerUpdate {
	return &RestLayerUpdate{service: service, logger: logger}
}

func (handler *RestLayerUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.LayerUpdate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestLayerUpdateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.LayerUpdateRequest{
		OwnerID:     ownerID,
		ID:          request.ID,
		Revision:    request.Revision,
		Name:        request.Name,
		Description: request.Description,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadLayer(result))
}
