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

// RestSceneDeleteService is the core operation [RestSceneDelete] delegates to.
type RestSceneDeleteService interface {
	Exec(ctx context.Context, request *core.SceneDeleteRequest) ([]*core.Scene, error)
}

// RestSceneDeleteRequest is the query the delete-scenes endpoint accepts.
type RestSceneDeleteRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
	// IDs lists the scenes to delete, one id parameter each.
	IDs []uuid.UUID `schema:"id"`
}

// RestSceneDelete is the HTTP handler for the delete-scenes endpoint. It responds with the deleted scenes, as they
// were.
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

	result, err := handler.service.Exec(ctx, &core.SceneDeleteRequest{
		OwnerID: ownerID, ProjectID: request.ProjectID, IDs: request.IDs,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(scene *core.Scene, _ int) Scene { return loadScene(scene) })
	httpf.SendJSONStatus(w, span, http.StatusOK, response)
}
