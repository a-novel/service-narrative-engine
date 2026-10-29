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

// RestBeatsCreateService is the core operation [RestBeatsCreate] delegates to.
type RestBeatsCreateService interface {
	Exec(ctx context.Context, request *core.BeatsCreateRequest) ([]*core.Beat, error)
}

// RestBeatsCreateItem is one beat of a [RestBeatsCreateRequest].
type RestBeatsCreateItem struct {
	// Title names the beat in the composer.
	Title string `json:"title"`
	// Color is the beat's color in the composer, as a #rrggbb hex code in either case.
	Color string `json:"color"`
	// Description says what the beat is for, written for the author.
	Description string `json:"description"`
	// Prompt holds extra instructions for the model when it writes about the beat.
	Prompt string `json:"prompt"`
}

// RestBeatsCreateRequest is the JSON body the create-beats endpoint accepts.
type RestBeatsCreateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Position is the place of the first new beat in narrative order, from 0, and the others follow it.
	// A position past the last beat appends them.
	Position int `json:"position"`

	// Beats lists the beats to add, in narrative order.
	Beats []RestBeatsCreateItem `json:"beats"`
}

// RestBeatsCreate is the HTTP handler for the create-beats endpoint. It responds 201 with the new beats in
// narrative order.
type RestBeatsCreate struct {
	service RestBeatsCreateService
	logger  logging.Log
}

func NewRestBeatsCreate(service RestBeatsCreateService, logger logging.Log) *RestBeatsCreate {
	return &RestBeatsCreate{service: service, logger: logger}
}

func (handler *RestBeatsCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.BeatsCreate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestBeatsCreateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.BeatsCreateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Position:  request.Position,
		Beats: lo.Map(request.Beats, func(item RestBeatsCreateItem, _ int) core.BeatsCreateItem {
			return loadBeatsCreateItem(item)
		}),
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(beat *core.Beat, _ int) Beat { return loadBeat(beat) })
	httpf.SendJSONStatus(w, span, http.StatusCreated, response)
}

// loadBeatsCreateItem converts one item of the request body to its core counterpart.
func loadBeatsCreateItem(item RestBeatsCreateItem) core.BeatsCreateItem {
	return core.BeatsCreateItem{Title: item.Title, Color: item.Color, Description: item.Description, Prompt: item.Prompt}
}
