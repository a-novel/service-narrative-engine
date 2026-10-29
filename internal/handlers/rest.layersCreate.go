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

// RestLayersCreateService is the core operation [RestLayersCreate] delegates to.
type RestLayersCreateService interface {
	Exec(ctx context.Context, request *core.LayersCreateRequest) ([]*core.Layer, error)
}

// RestLayersCreateItem is one layer of a [RestLayersCreateRequest].
type RestLayersCreateItem struct {
	// Kind names the content schema the layer's cells follow, such as manuscript or notes.
	Kind string `json:"kind"`

	// Name labels the layer's row in the composer.
	Name string `json:"name"`
	// Description says what the layer records.
	Description string `json:"description"`
}

// RestLayersCreateRequest is the JSON body the create-layers endpoint accepts.
type RestLayersCreateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Position is the place of the first new layer among the project's rows, from 0, and the others
	// follow it. A position past the last layer appends them.
	Position int `json:"position"`

	// Layers lists the layers to add, in order.
	Layers []RestLayersCreateItem `json:"layers"`
}

// RestLayersCreate is the HTTP handler for the create-layers endpoint. It responds 201 with the new layers in
// order.
type RestLayersCreate struct {
	service RestLayersCreateService
	logger  logging.Log
}

func NewRestLayersCreate(service RestLayersCreateService, logger logging.Log) *RestLayersCreate {
	return &RestLayersCreate{service: service, logger: logger}
}

func (handler *RestLayersCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.LayersCreate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestLayersCreateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.LayersCreateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Position:  request.Position,
		Layers: lo.Map(request.Layers, func(item RestLayersCreateItem, _ int) core.LayersCreateItem {
			return loadLayersCreateItem(item)
		}),
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(layer *core.Layer, _ int) Layer { return loadLayer(layer) })
	httpf.SendJSONStatus(w, span, http.StatusCreated, response)
}

// loadLayersCreateItem converts one item of the request body to its core counterpart.
func loadLayersCreateItem(item RestLayersCreateItem) core.LayersCreateItem {
	return core.LayersCreateItem{Kind: item.Kind, Name: item.Name, Description: item.Description}
}
