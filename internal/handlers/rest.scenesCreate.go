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
	// BeatID is the beat the scene joins.
	BeatID uuid.UUID `json:"beatID"`

	// Position is the scene's place in its beat, from 0: before the beat's scene at that position, or last
	// past the end. New scenes at one position keep the order of the request.
	Position int `json:"position"`

	// Title names the scene in the composer.
	Title string `json:"title"`
	// Description refines the beat's description for the scene. Empty keeps the beat's.
	Description string `json:"description"`
	// Prompt refines the beat's prompt for the scene. Empty keeps the beat's.
	Prompt string `json:"prompt"`
}

// RestScenesCreateRequest is the JSON body the create-scenes endpoint accepts.
type RestScenesCreateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Scenes lists the scenes to add, each with its beat and its place in it.
	Scenes []RestScenesCreateItem `json:"scenes"`
}

// RestScenesCreate is the HTTP handler for the create-scenes endpoint. It responds 201 with the new scenes in
// the order of the request.
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

	serviceRequest := &core.ScenesCreateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Scenes: lo.Map(request.Scenes, func(item RestScenesCreateItem, _ int) core.ScenesCreateItem {
			return loadScenesCreateItem(item)
		}),
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	response := lo.Map(result, func(scene *core.Scene, _ int) Scene { return loadScene(scene) })
	httpf.SendJSONStatus(w, span, http.StatusCreated, response)
}

// loadScenesCreateItem converts one item of the request body to its core counterpart.
func loadScenesCreateItem(item RestScenesCreateItem) core.ScenesCreateItem {
	return core.ScenesCreateItem{
		BeatID: item.BeatID, Position: item.Position, Title: item.Title, Description: item.Description,
		Prompt: item.Prompt,
	}
}
