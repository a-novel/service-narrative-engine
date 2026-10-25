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

// RestSceneDeleteService is the core operation [RestSceneDelete] delegates to.
type RestSceneDeleteService interface {
	Exec(ctx context.Context, request *core.SceneDeleteRequest) (*core.Scene, error)
}

// RestSceneDeleteRequest is the query parameters the delete-scene endpoint accepts.
type RestSceneDeleteRequest struct {
	ID       uuid.UUID `schema:"id"`
	Revision int       `schema:"revision"`
}

// RestSceneDelete is the HTTP handler for the delete-scene endpoint. It responds with the deleted
// scene.
type RestSceneDelete struct {
	service RestSceneDeleteService
	logger  logging.Log
}

func NewRestSceneDelete(service RestSceneDeleteService, logger logging.Log) *RestSceneDelete {
	return &RestSceneDelete{service: service, logger: logger}
}

func (handler *RestSceneDelete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.SceneDelete")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestSceneDeleteRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.SceneDeleteRequest{
		OwnerID:  ownerID,
		ID:       request.ID,
		Revision: request.Revision,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadScene(result))
}
