package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestIdeaVersionCreateService saves one complete Idea Version.
type RestIdeaVersionCreateService interface {
	Exec(ctx context.Context, request *core.IdeaVersionCreateRequest) (*core.Idea, error)
}

// RestIdeaVersionCreateRequest contains one complete static Idea value.
type RestIdeaVersionCreateRequest struct {
	Value RestIdeaValue `json:"value"`
}

// RestIdeaVersionCreate publishes Idea version writes over HTTP.
type RestIdeaVersionCreate struct {
	service RestIdeaVersionCreateService
	logger  logging.Log
}

// NewRestIdeaVersionCreate creates an Idea Version handler.
func NewRestIdeaVersionCreate(
	service RestIdeaVersionCreateService,
	logger logging.Log,
) *RestIdeaVersionCreate {
	return &RestIdeaVersionCreate{service: service, logger: logger}
}

// ServeHTTP appends one Idea Version under an owned Project.
func (handler *RestIdeaVersionCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.IdeaVersionCreate")
	defer span.End()

	actor, projectID, err := restProjectIdentity(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restTransportErrors, err)

		return
	}

	var request RestIdeaVersionCreateRequest

	err = decodeRestJSON(r, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restTransportErrors, err)

		return
	}

	idea, err := handler.service.Exec(ctx, &core.IdeaVersionCreateRequest{
		Actor:     *actor,
		ProjectID: projectID,
		Title:     request.Value.Title,
		Genre:     request.Value.Genre,
		Seed:      request.Value.Seed,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restServiceErrors, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusCreated, loadRestIdeaVersion(idea))
}
