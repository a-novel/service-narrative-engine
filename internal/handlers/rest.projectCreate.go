package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestProjectCreateService creates a Project and its initial Idea.
type RestProjectCreateService interface {
	Exec(ctx context.Context, request *core.IdeaCreateRequest) (*core.Idea, error)
}

// RestProjectCreateRequest contains the complete initial Idea.
type RestProjectCreateRequest struct {
	Idea RestIdeaValue `json:"idea"`
}

// RestProjectCreate publishes Project creation over HTTP.
type RestProjectCreate struct {
	service RestProjectCreateService
	logger  logging.Log
}

// NewRestProjectCreate creates a Project create handler.
func NewRestProjectCreate(
	service RestProjectCreateService,
	logger logging.Log,
) *RestProjectCreate {
	return &RestProjectCreate{service: service, logger: logger}
}

// ServeHTTP creates an owned Project and returns its opening snapshot.
func (handler *RestProjectCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectCreate")
	defer span.End()

	actor, err := ActorFromContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restServiceErrors, err)

		return
	}

	var request RestProjectCreateRequest

	err = decodeRestJSON(r, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restTransportErrors, err)

		return
	}

	idea, err := handler.service.Exec(ctx, &core.IdeaCreateRequest{
		Actor: *actor,
		Title: request.Idea.Title,
		Genre: request.Idea.Genre,
		Seed:  request.Idea.Seed,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restServiceErrors, err)

		return
	}

	w.Header().Set("Location", "/v0/projects/"+idea.ProjectID.String())
	httpf.SendJSONStatus(ctx, w, span, http.StatusCreated, &RestProject{
		ID:         idea.ProjectID,
		CreatedAt:  idea.ProjectCreatedAt,
		Idea:       loadRestIdeaVersion(idea),
		StepValues: make([]*RestStepValueVersion, 0),
	})
}
