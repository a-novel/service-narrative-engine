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

// RestBeatDeleteService is the core operation [RestBeatDelete] delegates to.
type RestBeatDeleteService interface {
	Exec(ctx context.Context, request *core.BeatDeleteRequest) (*core.Beat, error)
}

// RestBeatDeleteRequest is the query parameters the delete-beat endpoint accepts.
type RestBeatDeleteRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
	ID        uuid.UUID `schema:"id"`
}

// RestBeatDelete is the HTTP handler for the delete-beat endpoint. It responds with the deleted
// beat.
type RestBeatDelete struct {
	service RestBeatDeleteService
	logger  logging.Log
}

func NewRestBeatDelete(service RestBeatDeleteService, logger logging.Log) *RestBeatDelete {
	return &RestBeatDelete{service: service, logger: logger}
}

func (handler *RestBeatDelete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.BeatDelete")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestBeatDeleteRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.BeatDeleteRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		ID:        request.ID,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadBeat(result))
}
