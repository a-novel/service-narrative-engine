package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestProjectCreateService is the core operation [RestProjectCreate] delegates to.
type RestProjectCreateService interface {
	Exec(ctx context.Context, request *core.ProjectCreateRequest) (*core.ProjectComposer, error)
}

// RestProjectCreateRequest is the JSON body the create-project endpoint accepts.
type RestProjectCreateRequest struct {
	// Title names the narrative.
	Title string `json:"title"`
	// Premise says what the narrative is about, in a few sentences.
	Premise string `json:"premise"`
	// Language is the language the project is written in: en or fr.
	Language string `json:"language"`

	// Scenes are the scenes the project starts with, in order.
	Scenes []RestProjectCreateScene `json:"scenes"`
}

// RestProjectCreate is the HTTP handler for the create-project endpoint. It responds 201 with the
// project, its manuscript layer and its first scenes.
type RestProjectCreate struct {
	service RestProjectCreateService
	logger  logging.Log
}

func NewRestProjectCreate(service RestProjectCreateService, logger logging.Log) *RestProjectCreate {
	return &RestProjectCreate{service: service, logger: logger}
}

func (handler *RestProjectCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectCreate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectCreateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.ProjectCreateRequest{
		OwnerID:  ownerID,
		Title:    request.Title,
		Premise:  request.Premise,
		Language: request.Language,
		Scenes:   lo.Map(request.Scenes, loadProjectCreateScene),
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusCreated, loadProjectComposer(result))
}

// RestProjectCreateScene is a scene a new project starts with, such as one a preset seeds.
type RestProjectCreateScene struct {
	// Title names the scene in the composer.
	Title string `json:"title"`
	// Color is the scene's color in the composer, as a #rrggbb hex code in either case.
	Color string `json:"color"`
	// Description says what the scene is for, written for the author.
	Description string `json:"description"`
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt string `json:"prompt"`
}

// loadProjectCreateScene adapts a seed scene of the request to lo.Map.
func loadProjectCreateScene(scene RestProjectCreateScene, _ int) core.ProjectCreateScene {
	return core.ProjectCreateScene{
		Title: scene.Title, Color: scene.Color, Description: scene.Description, Prompt: scene.Prompt,
	}
}
