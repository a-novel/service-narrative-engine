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

// RestLayerCreateService is the core operation [RestLayerCreate] delegates to.
type RestLayerCreateService interface {
	Exec(ctx context.Context, request *core.LayerCreateRequest) (*core.Layer, error)
}

// RestLayerCreateRequest is the JSON body the create-layer endpoint accepts.
type RestLayerCreateRequest struct {
	ProjectID   uuid.UUID `json:"projectID"`
	After       uuid.UUID `json:"after"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// RestLayerCreate is the HTTP handler for the create-layer endpoint. It responds 201 with the notes
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
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.LayerCreateRequest{
		OwnerID:     ownerID,
		ProjectID:   request.ProjectID,
		After:       request.After,
		Name:        request.Name,
		Description: request.Description,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusCreated, loadLayer(result))
}
