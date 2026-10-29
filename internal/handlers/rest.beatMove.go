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

// RestBeatMoveService is the core operation [RestBeatMove] delegates to.
type RestBeatMoveService interface {
	Exec(ctx context.Context, request *core.BeatMoveRequest) (*core.Beat, error)
}

// RestBeatMoveRequest is the JSON body the move-beat endpoint accepts.
type RestBeatMoveRequest struct {
	ProjectID uuid.UUID `json:"projectID"`
	ID        uuid.UUID `json:"id"`

	// Position is the beat's place in narrative order after the move, from 0. A position past the last
	// beat moves it last.
	Position int `json:"position"`
}

// RestBeatMove is the HTTP handler for the move-beat endpoint. It responds with the beat at its new
// position.
type RestBeatMove struct {
	service RestBeatMoveService
	logger  logging.Log
}

func NewRestBeatMove(service RestBeatMoveService, logger logging.Log) *RestBeatMove {
	return &RestBeatMove{service: service, logger: logger}
}

func (handler *RestBeatMove) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.BeatMove")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestBeatMoveRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.BeatMoveRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		ID:        request.ID,
		Position:  request.Position,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadBeat(result))
}
