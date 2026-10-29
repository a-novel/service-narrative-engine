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

// RestBeatsDeleteService is the core operation [RestBeatsDelete] delegates to.
type RestBeatsDeleteService interface {
	Exec(ctx context.Context, request *core.BeatsDeleteRequest) ([]*core.Beat, error)
}

// RestBeatsDeleteRequest is the query the delete-beats endpoint accepts.
type RestBeatsDeleteRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
	// IDs lists the beats to delete, one id parameter each.
	IDs []uuid.UUID `schema:"id"`
}

// RestBeatsDelete is the HTTP handler for the delete-beats endpoint. It responds with the deleted beats, as they
// were.
type RestBeatsDelete struct {
	service RestBeatsDeleteService
	logger  logging.Log
}

func NewRestBeatsDelete(service RestBeatsDeleteService, logger logging.Log) *RestBeatsDelete {
	return &RestBeatsDelete{service: service, logger: logger}
}

func (handler *RestBeatsDelete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.BeatsDelete")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestBeatsDeleteRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.BeatsDeleteRequest{
		OwnerID: ownerID, ProjectID: request.ProjectID, IDs: request.IDs,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(beat *core.Beat, _ int) Beat { return loadBeat(beat) })
	httpf.SendJSONStatus(w, span, http.StatusOK, response)
}
