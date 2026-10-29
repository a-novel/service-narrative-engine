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

// RestScenesUpdateService is the core operation [RestScenesUpdate] delegates to.
type RestScenesUpdateService interface {
	Exec(ctx context.Context, request *core.ScenesUpdateRequest) ([]*core.Scene, error)
}

// RestScenesUpdateItem is one scene of a [RestScenesUpdateRequest].
type RestScenesUpdateItem struct {
	ID uuid.UUID `json:"id"`

	// Title names the scene in the composer.
	Title string `json:"title"`
	// Color is the scene's color in the composer, as a #rrggbb hex code in either case.
	Color string `json:"color"`
	// Description says what the scene is for, written for the author.
	Description string `json:"description"`
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt string `json:"prompt"`
}

// RestScenesUpdateRequest is the JSON body the update-scenes endpoint accepts.
type RestScenesUpdateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Scenes lists the scenes to update, each once.
	Scenes []RestScenesUpdateItem `json:"scenes"`
}

// RestScenesUpdate is the HTTP handler for the update-scenes endpoint. It responds with the updated scenes, in
// the order of the request.
type RestScenesUpdate struct {
	service RestScenesUpdateService
	logger  logging.Log
}

func NewRestScenesUpdate(service RestScenesUpdateService, logger logging.Log) *RestScenesUpdate {
	return &RestScenesUpdate{service: service, logger: logger}
}

func (handler *RestScenesUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ScenesUpdate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestScenesUpdateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.ScenesUpdateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Scenes: lo.Map(request.Scenes, func(item RestScenesUpdateItem, _ int) core.ScenesUpdateItem {
			return loadScenesUpdateItem(item)
		}),
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(scene *core.Scene, _ int) Scene { return loadScene(scene) })
	httpf.SendJSONStatus(w, span, http.StatusOK, response)
}

// loadScenesUpdateItem converts one item of the request body to its core counterpart.
func loadScenesUpdateItem(item RestScenesUpdateItem) core.ScenesUpdateItem {
	return core.ScenesUpdateItem{
		ID: item.ID, Title: item.Title, Color: item.Color, Description: item.Description, Prompt: item.Prompt,
	}
}
