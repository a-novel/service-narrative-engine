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

// RestSceneListService is the core operation [RestSceneList] delegates to.
type RestSceneListService interface {
	Exec(ctx context.Context, request *core.SceneListRequest) (*core.ScenePage, error)
}

// RestSceneListRequest is the query parameters the list-scenes endpoint accepts.
type RestSceneListRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`

	// Limit caps the page. Zero takes the default size.
	Limit int `schema:"limit"`
	// Offset is how many scenes come before the page, in narrative order.
	Offset int `schema:"offset"`
}

// RestSceneList is the HTTP handler for the list-scenes endpoint. It responds with a page of a
// project's scenes, in narrative order, and their total.
type RestSceneList struct {
	service RestSceneListService
	logger  logging.Log
}

func NewRestSceneList(service RestSceneListService, logger logging.Log) *RestSceneList {
	return &RestSceneList{service: service, logger: logger}
}

func (handler *RestSceneList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.SceneList")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestSceneListRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.SceneListRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Limit:     request.Limit,
		Offset:    request.Offset,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadScenePage(result))
}
