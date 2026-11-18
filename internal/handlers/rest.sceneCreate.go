package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestSceneCreateService is the batch operation [RestSceneCreate] delegates to, with a list of one.
type RestSceneCreateService interface {
	Exec(ctx context.Context, request *core.ScenesCreateRequest) ([]*core.Scene, error)
}

// RestSceneCreateRequest is the JSON body the create-scene endpoint accepts.
type RestSceneCreateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`
	// BeatID is the beat the scene joins.
	BeatID uuid.UUID `json:"beatID"`

	// Position is the scene's place in its beat, from 0. A position past the beat's last scene appends it.
	Position int `json:"position"`

	// Title names the scene in the composer.
	Title string `json:"title"`
	// Description refines the beat's description for the scene. Empty keeps the beat's.
	Description string `json:"description"`
	// Prompt refines the beat's prompt for the scene. Empty keeps the beat's.
	Prompt string `json:"prompt"`
}

// RestSceneCreate is the HTTP handler for the create-scene endpoint. It responds 200 with the scene at
// its place.
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

	ownerID, err := serviceauthentication.RequestUser(r.Context())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestSceneCreateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	span.SetAttributes(
		attribute.String("beat.id", request.BeatID.String()),
		attribute.Int("scene.position", request.Position),
	)

	serviceRequest := &core.ScenesCreateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Scenes: []core.ScenesCreateItem{{
			BeatID:      request.BeatID,
			Position:    request.Position,
			Title:       request.Title,
			Description: request.Description,
			Prompt:      request.Prompt,
		}},
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, tagLimit(itemFields(err)))

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadScene(result[0]))
}
