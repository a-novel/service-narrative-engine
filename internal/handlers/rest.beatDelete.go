package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestBeatDeleteService is the batch operation [RestBeatDelete] delegates to, with a list of one.
type RestBeatDeleteService interface {
	Exec(ctx context.Context, request *core.BeatsDeleteRequest) ([]*core.Beat, error)
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
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	span.SetAttributes(
		attribute.String("beat.id", request.ID.String()),
	)

	serviceRequest := &core.BeatsDeleteRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		IDs:       []uuid.UUID{request.ID},
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, itemFields(err))

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadBeat(result[0]))
}
