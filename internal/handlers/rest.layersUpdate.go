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

// RestLayersUpdateService is the core operation [RestLayersUpdate] delegates to.
type RestLayersUpdateService interface {
	Exec(ctx context.Context, request *core.LayersUpdateRequest) ([]*core.Layer, error)
}

// RestLayersUpdateItem is one layer of a [RestLayersUpdateRequest].
type RestLayersUpdateItem struct {
	ID uuid.UUID `json:"id"`

	// Name labels the layer's row in the composer.
	Name string `json:"name"`
	// Description says what the layer records.
	Description string `json:"description"`
}

// RestLayersUpdateRequest is the JSON body the update-layers endpoint accepts.
type RestLayersUpdateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Layers lists the layers to update, each once.
	Layers []RestLayersUpdateItem `json:"layers"`
}

// RestLayersUpdate is the HTTP handler for the update-layers endpoint. It responds with the updated layers, in
// the order of the request.
type RestLayersUpdate struct {
	service RestLayersUpdateService
	logger  logging.Log
}

func NewRestLayersUpdate(service RestLayersUpdateService, logger logging.Log) *RestLayersUpdate {
	return &RestLayersUpdate{service: service, logger: logger}
}

func (handler *RestLayersUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.LayersUpdate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestLayersUpdateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.LayersUpdateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Layers: lo.Map(request.Layers, func(item RestLayersUpdateItem, _ int) core.LayersUpdateItem {
			return loadLayersUpdateItem(item)
		}),
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(layer *core.Layer, _ int) Layer { return loadLayer(layer) })
	httpf.SendJSONStatus(w, span, http.StatusOK, response)
}

// loadLayersUpdateItem converts one item of the request body to its core counterpart.
func loadLayersUpdateItem(item RestLayersUpdateItem) core.LayersUpdateItem {
	return core.LayersUpdateItem{ID: item.ID, Name: item.Name, Description: item.Description}
}
