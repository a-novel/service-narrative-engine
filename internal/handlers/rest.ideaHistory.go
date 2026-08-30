package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestIdeaHistoryService reads one Project's retained Idea Versions.
type RestIdeaHistoryService interface {
	Exec(ctx context.Context, request *core.IdeaHistoryRequest) ([]*core.Idea, error)
}

// RestIdeaHistory publishes bounded Idea history over HTTP.
type RestIdeaHistory struct {
	service RestIdeaHistoryService
	logger  logging.Log
}

// NewRestIdeaHistory creates an Idea history handler.
func NewRestIdeaHistory(service RestIdeaHistoryService, logger logging.Log) *RestIdeaHistory {
	return &RestIdeaHistory{service: service, logger: logger}
}

// ServeHTTP returns retained Idea Versions newest first.
func (handler *RestIdeaHistory) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.IdeaHistory")
	defer span.End()

	actor, projectID, err := restProjectIdentity(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restTransportErrors, err)

		return
	}

	ideas, err := handler.service.Exec(ctx, &core.IdeaHistoryRequest{
		Actor:     *actor,
		ProjectID: projectID,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restServiceErrors, err)

		return
	}

	response := make([]*RestIdeaVersion, 0, len(ideas))
	for _, idea := range ideas {
		response = append(response, loadRestIdeaVersion(idea))
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, response)
}
