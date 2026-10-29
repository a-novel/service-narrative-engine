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

// RestBeatUpdateService is the core operation [RestBeatUpdate] delegates to.
type RestBeatUpdateService interface {
	Exec(ctx context.Context, request *core.BeatUpdateRequest) (*core.Beat, error)
}

// RestBeatUpdateRequest is the JSON body the update-beat endpoint accepts.
type RestBeatUpdateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`
	ID        uuid.UUID `json:"id"`

	// Title names the beat in the composer.
	Title string `json:"title"`
	// Color is the beat's color in the composer, as a #rrggbb hex code in either case.
	Color string `json:"color"`
	// Description says what the beat is for, written for the author.
	Description string `json:"description"`
	// Prompt holds extra instructions for the model when it writes about the beat.
	Prompt string `json:"prompt"`
}

// RestBeatUpdate is the HTTP handler for the update-beat endpoint. It responds with the updated
// beat.
type RestBeatUpdate struct {
	service RestBeatUpdateService
	logger  logging.Log
}

func NewRestBeatUpdate(service RestBeatUpdateService, logger logging.Log) *RestBeatUpdate {
	return &RestBeatUpdate{service: service, logger: logger}
}

func (handler *RestBeatUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.BeatUpdate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestBeatUpdateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.BeatUpdateRequest{
		OwnerID:     ownerID,
		ProjectID:   request.ProjectID,
		ID:          request.ID,
		Title:       request.Title,
		Color:       request.Color,
		Description: request.Description,
		Prompt:      request.Prompt,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadBeat(result))
}
