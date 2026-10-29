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

// RestBeatsUpdateService is the core operation [RestBeatsUpdate] delegates to.
type RestBeatsUpdateService interface {
	Exec(ctx context.Context, request *core.BeatsUpdateRequest) ([]*core.Beat, error)
}

// RestBeatsUpdateItem is one beat of a [RestBeatsUpdateRequest].
type RestBeatsUpdateItem struct {
	ID uuid.UUID `json:"id"`

	// Title names the beat in the composer.
	Title string `json:"title"`
	// Color is the beat's color in the composer, as a #rrggbb hex code in either case.
	Color string `json:"color"`
	// Description says what the beat is for, written for the author.
	Description string `json:"description"`
	// Prompt holds extra instructions for the model when it writes about the beat.
	Prompt string `json:"prompt"`
}

// RestBeatsUpdateRequest is the JSON body the update-beats endpoint accepts.
type RestBeatsUpdateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Beats lists the beats to update, each once.
	Beats []RestBeatsUpdateItem `json:"beats"`
}

// RestBeatsUpdate is the HTTP handler for the update-beats endpoint. It responds with the updated beats, in
// the order of the request.
type RestBeatsUpdate struct {
	service RestBeatsUpdateService
	logger  logging.Log
}

func NewRestBeatsUpdate(service RestBeatsUpdateService, logger logging.Log) *RestBeatsUpdate {
	return &RestBeatsUpdate{service: service, logger: logger}
}

func (handler *RestBeatsUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.BeatsUpdate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestBeatsUpdateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.BeatsUpdateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Beats: lo.Map(request.Beats, func(item RestBeatsUpdateItem, _ int) core.BeatsUpdateItem {
			return loadBeatsUpdateItem(item)
		}),
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(beat *core.Beat, _ int) Beat { return loadBeat(beat) })
	httpf.SendJSONStatus(w, span, http.StatusOK, response)
}

// loadBeatsUpdateItem converts one item of the request body to its core counterpart.
func loadBeatsUpdateItem(item RestBeatsUpdateItem) core.BeatsUpdateItem {
	return core.BeatsUpdateItem{
		ID: item.ID, Title: item.Title, Color: item.Color, Description: item.Description, Prompt: item.Prompt,
	}
}
