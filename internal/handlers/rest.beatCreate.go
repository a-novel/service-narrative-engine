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

// RestBeatCreateService is the core operation [RestBeatCreate] delegates to.
type RestBeatCreateService interface {
	Exec(ctx context.Context, request *core.BeatCreateRequest) (*core.Beat, error)
}

// RestBeatCreateRequest is the JSON body the create-beat endpoint accepts.
type RestBeatCreateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Position is the beat's place in narrative order, from 0. A position past the last beat appends
	// it.
	Position int `json:"position"`

	// Title names the beat in the composer.
	Title string `json:"title"`
	// Color is the beat's color in the composer, as a #rrggbb hex code in either case.
	Color string `json:"color"`
	// Description says what the beat is for, written for the author.
	Description string `json:"description"`
	// Prompt holds extra instructions for the model when it writes about the beat.
	Prompt string `json:"prompt"`
}

// RestBeatCreate is the HTTP handler for the create-beat endpoint. It responds 201 with the beat at
// its place.
type RestBeatCreate struct {
	service RestBeatCreateService
	logger  logging.Log
}

func NewRestBeatCreate(service RestBeatCreateService, logger logging.Log) *RestBeatCreate {
	return &RestBeatCreate{service: service, logger: logger}
}

func (handler *RestBeatCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.BeatCreate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestBeatCreateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.BeatCreateRequest{
		OwnerID:     ownerID,
		ProjectID:   request.ProjectID,
		Position:    request.Position,
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

	httpf.SendJSONStatus(w, span, http.StatusCreated, loadBeat(result))
}
