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

// RestSceneCreateService is the core operation [RestSceneCreate] delegates to.
type RestSceneCreateService interface {
	Exec(ctx context.Context, request *core.SceneCreateRequest) (*core.Scene, error)
}

// RestSceneCreateRequest is the JSON body the create-scene endpoint accepts.
type RestSceneCreateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`
	After     uuid.UUID `json:"after"`
	Title     string    `json:"title"`
	Synopsis  string    `json:"synopsis"`
}

// RestSceneCreate is the HTTP handler for the create-scene endpoint. It responds 201 with the scene at
// its place.
type RestSceneCreate struct {
	service RestSceneCreateService
	logger  logging.Log
}

func NewRestSceneCreate(service RestSceneCreateService, logger logging.Log) *RestSceneCreate {
	return &RestSceneCreate{service: service, logger: logger}
}

func (handler *RestSceneCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.SceneCreate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestSceneCreateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.SceneCreateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		After:     request.After,
		Title:     request.Title,
		Synopsis:  request.Synopsis,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusCreated, loadScene(result))
}
