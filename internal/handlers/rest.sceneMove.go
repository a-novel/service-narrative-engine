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

// RestSceneMoveService is the core operation [RestSceneMove] delegates to.
type RestSceneMoveService interface {
	Exec(ctx context.Context, request *core.SceneMoveRequest) (*core.Scene, error)
}

// RestSceneMoveRequest is the JSON body the move-scene endpoint accepts.
type RestSceneMoveRequest struct {
	ID    uuid.UUID `json:"id"`
	After uuid.UUID `json:"after"`
}

// RestSceneMove is the HTTP handler for the move-scene endpoint. It responds with the scene at its new
// position.
type RestSceneMove struct {
	service RestSceneMoveService
	logger  logging.Log
}

func NewRestSceneMove(service RestSceneMoveService, logger logging.Log) *RestSceneMove {
	return &RestSceneMove{service: service, logger: logger}
}

func (handler *RestSceneMove) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.SceneMove")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestSceneMoveRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.SceneMoveRequest{
		OwnerID: ownerID,
		ID:      request.ID,
		After:   request.After,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadScene(result))
}
