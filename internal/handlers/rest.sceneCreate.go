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

// RestSceneCreateService is the core operation [RestSceneCreate] delegates to.
type RestSceneCreateService interface {
	Exec(ctx context.Context, request *core.SceneCreateRequest) ([]*core.Scene, error)
}

// RestSceneCreateItem is one scene of a [RestSceneCreateRequest].
type RestSceneCreateItem struct {
	// Title names the scene in the composer.
	Title string `json:"title"`
	// Color is the scene's color in the composer, as a #rrggbb hex code in either case.
	Color string `json:"color"`
	// Description says what the scene is for, written for the author.
	Description string `json:"description"`
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt string `json:"prompt"`
}

// RestSceneCreateRequest is the JSON body the create-scenes endpoint accepts.
type RestSceneCreateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Position is the place of the first new scene in narrative order, from 0, and the others follow it.
	// A position past the last scene appends them.
	Position int `json:"position"`
	// Scenes lists the scenes to add, in narrative order.
	Scenes []RestSceneCreateItem `json:"scenes"`
}

// RestSceneCreate is the HTTP handler for the create-scenes endpoint. It responds 201 with the new scenes in
// narrative order.
type RestSceneCreate struct {
	service RestSceneCreateService
	logger  logging.Log
}

func NewRestSceneCreate(service RestSceneCreateService, logger logging.Log) *RestSceneCreate {
	return &RestSceneCreate{service: service, logger: logger}
}

func (handler *RestSceneCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.SceneCreate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestSceneCreateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.SceneCreateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Position:  request.Position,
		Scenes: lo.Map(request.Scenes, func(item RestSceneCreateItem, _ int) core.SceneCreateItem {
			return core.SceneCreateItem{
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
