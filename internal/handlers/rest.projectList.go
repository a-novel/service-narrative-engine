package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestProjectListService is the core operation [RestProjectList] delegates to.
type RestProjectListService interface {
	Exec(ctx context.Context, request *core.ProjectListRequest) (*core.ProjectPage, error)
}

// RestProjectListRequest is the query parameters the list-projects endpoint accepts.
type RestProjectListRequest struct {
	// Limit caps the page. Zero takes the default size.
	Limit int `schema:"limit"`
	// Offset is how many projects come before the page, newest first.
	Offset int `schema:"offset"`
}

// RestProjectList is the HTTP handler for the list-projects endpoint. It responds with a page of the
// caller's projects, newest first, and their total.
type RestProjectList struct {
	service RestProjectListService
	logger  logging.Log
}

func NewRestProjectList(service RestProjectListService, logger logging.Log) *RestProjectList {
	return &RestProjectList{service: service, logger: logger}
}

func (handler *RestProjectList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectList")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectListRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.ProjectListRequest{
		OwnerID: ownerID,
		Limit:   request.Limit,
		Offset:  request.Offset,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadProjectPage(result))
}
