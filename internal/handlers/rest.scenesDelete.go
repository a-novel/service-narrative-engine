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

// RestScenesDeleteService is the core operation [RestScenesDelete] delegates to.
type RestScenesDeleteService interface {
	Exec(ctx context.Context, request *core.ScenesDeleteRequest) ([]*core.Scene, error)
}

// RestScenesDeleteRequest is the query the delete-scenes endpoint accepts.
type RestScenesDeleteRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
	// IDs lists the scenes to delete, one id parameter each.
	IDs []uuid.UUID `schema:"id"`
}

// RestScenesDelete is the HTTP handler for the delete-scenes endpoint. It responds with the deleted scenes, as they
// were.
type RestScenesDelete struct {
	service RestScenesDeleteService
	logger  logging.Log
}

func NewRestScenesDelete(service RestScenesDeleteService, logger logging.Log) *RestScenesDelete {
	return &RestScenesDelete{service: service, logger: logger}
}

func (handler *RestScenesDelete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ScenesDelete")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestScenesDeleteRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.ScenesDeleteRequest{
		OwnerID: ownerID, ProjectID: request.ProjectID, IDs: request.IDs,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(scene *core.Scene, _ int) Scene { return loadScene(scene) })
	httpf.SendJSONStatus(w, span, http.StatusOK, response)
}
