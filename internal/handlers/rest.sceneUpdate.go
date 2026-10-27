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

// RestSceneUpdateService is the core operation [RestSceneUpdate] delegates to.
type RestSceneUpdateService interface {
	Exec(ctx context.Context, request *core.SceneUpdateRequest) (*core.Scene, error)
}

// RestSceneUpdateRequest is the JSON body the update-scene endpoint accepts.
type RestSceneUpdateRequest struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Color       string    `json:"color"`
	Description string    `json:"description"`
	Prompt      string    `json:"prompt"`
}

// RestSceneUpdate is the HTTP handler for the update-scene endpoint. It responds with the updated
// scene.
type RestSceneUpdate struct {
	service RestSceneUpdateService
	logger  logging.Log
}

func NewRestSceneUpdate(service RestSceneUpdateService, logger logging.Log) *RestSceneUpdate {
	return &RestSceneUpdate{service: service, logger: logger}
}

func (handler *RestSceneUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.SceneUpdate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestSceneUpdateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.SceneUpdateRequest{
		OwnerID:     ownerID,
		ID:          request.ID,
		Title:       request.Title,
		Color:       request.Color,
		Description: request.Description,
		Prompt:      request.Prompt,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadScene(result))
}
