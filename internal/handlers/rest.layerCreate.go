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

// RestLayerCreateService is the batch operation [RestLayerCreate] delegates to, with a list of one.
type RestLayerCreateService interface {
	Exec(ctx context.Context, request *core.LayersCreateRequest) ([]*core.Layer, error)
}

// RestLayerCreateRequest is the JSON body the create-layer endpoint accepts.
type RestLayerCreateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Position is the layer's place among the project's rows, from 0. A position past the last layer
	// appends it.
	Position int `json:"position"`
	// Kind names the content schema the layer's cells follow, such as manuscript or notes.
	Kind string `json:"kind"`

	// Name labels the layer's row in the composer.
	Name string `json:"name"`
	// Description says what the layer records.
	Description string `json:"description"`
}

// RestLayerCreate is the HTTP handler for the create-layer endpoint. It responds 200 with the
// layer at its place.
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
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	serviceRequest := &core.LayersCreateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Position:  request.Position,
		Layers: []core.LayersCreateItem{{
			Kind:        request.Kind,
			Name:        request.Name,
			Description: request.Description,
		}},
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, tagLimit(err))

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadLayer(result[0]))
}
