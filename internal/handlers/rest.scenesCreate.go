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

// RestScenesCreateService is the core operation [RestScenesCreate] delegates to.
type RestScenesCreateService interface {
	Exec(ctx context.Context, request *core.ScenesCreateRequest) ([]*core.Scene, error)
}

// RestScenesCreateItem is one scene of a [RestScenesCreateRequest].
type RestScenesCreateItem struct {
	// Title names the scene in the composer.
	Title string `json:"title"`
	// Color is the scene's color in the composer, as a #rrggbb hex code in either case.
	Color string `json:"color"`
	// Description says what the scene is for, written for the author.
	Description string `json:"description"`
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt string `json:"prompt"`
}

// RestScenesCreateRequest is the JSON body the create-scenes endpoint accepts.
type RestScenesCreateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Position is the place of the first new scene in narrative order, from 0, and the others follow it.
	// A position past the last scene appends them.
	Position int `json:"position"`
	// Scenes lists the scenes to add, in narrative order.
	Scenes []RestScenesCreateItem `json:"scenes"`
}

// RestScenesCreate is the HTTP handler for the create-scenes endpoint. It responds 201 with the new scenes in
// narrative order.
type RestScenesCreate struct {
	service RestScenesCreateService
	logger  logging.Log
}

func NewRestScenesCreate(service RestScenesCreateService, logger logging.Log) *RestScenesCreate {
	return &RestScenesCreate{service: service, logger: logger}
}

func (handler *RestScenesCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ScenesCreate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestScenesCreateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.ScenesCreateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Position:  request.Position,
		Scenes: lo.Map(request.Scenes, func(item RestScenesCreateItem, _ int) core.ScenesCreateItem {
			return core.ScenesCreateItem{
				Title: item.Title, Color: item.Color, Description: item.Description, Prompt: item.Prompt,
			}
		}),
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(scene *core.Scene, _ int) Scene { return loadScene(scene) })
	httpf.SendJSONStatus(w, span, http.StatusCreated, response)
}
