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

// RestLayerCreateService is the core operation [RestLayerCreate] delegates to.
type RestLayerCreateService interface {
	Exec(ctx context.Context, request *core.LayerCreateRequest) ([]*core.Layer, error)
}

// RestLayerCreateItem is one layer of a [RestLayerCreateRequest].
type RestLayerCreateItem struct {
	// Kind names the content schema the layer's cells follow, such as manuscript or notes.
	Kind string `json:"kind"`

	// Name labels the layer's row in the composer.
	Name string `json:"name"`
	// Description says what the layer records.
	Description string `json:"description"`
}

// RestLayerCreateRequest is the JSON body the create-layers endpoint accepts.
type RestLayerCreateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Position is the place of the first new layer among the project's rows, from 0, and the others
	// follow it. A position past the last layer appends them.
	Position int `json:"position"`
	// Layers lists the layers to add, in order.
	Layers []RestLayerCreateItem `json:"layers"`
}

// RestLayerCreate is the HTTP handler for the create-layers endpoint. It responds 201 with the new layers in
// order.
type RestLayerCreate struct {
	service RestLayerCreateService
	logger  logging.Log
}

func NewRestLayerCreate(service RestLayerCreateService, logger logging.Log) *RestLayerCreate {
	return &RestLayerCreate{service: service, logger: logger}
}

func (handler *RestLayerCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.LayerCreate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestLayerCreateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.LayerCreateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Position:  request.Position,
		Layers: lo.Map(request.Layers, func(item RestLayerCreateItem, _ int) core.LayerCreateItem {
			return core.LayerCreateItem{Kind: item.Kind, Name: item.Name, Description: item.Description}
		}),
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(layer *core.Layer, _ int) Layer { return loadLayer(layer) })
	httpf.SendJSONStatus(w, span, http.StatusCreated, response)
}
