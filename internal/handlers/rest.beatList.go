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

// RestBeatListService is the core operation [RestBeatList] delegates to.
type RestBeatListService interface {
	Exec(ctx context.Context, request *core.BeatListRequest) (*core.BeatPage, error)
}

// RestBeatListRequest is the query parameters the list-beats endpoint accepts.
type RestBeatListRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`

	// Limit caps the page. Zero takes the default size.
	Limit int `schema:"limit"`
	// Offset is how many beats come before the page, in narrative order.
	Offset int `schema:"offset"`
}

// RestBeatList is the HTTP handler for the list-beats endpoint. It responds with a page of a
// project's beats, in narrative order, and their total.
type RestBeatList struct {
	service RestBeatListService
	logger  logging.Log
}

func NewRestBeatList(service RestBeatListService, logger logging.Log) *RestBeatList {
	return &RestBeatList{service: service, logger: logger}
}

func (handler *RestBeatList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.BeatList")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestBeatListRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.BeatListRequest{
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

	httpf.SendJSONStatus(w, span, http.StatusOK, loadBeatPage(result))
}
