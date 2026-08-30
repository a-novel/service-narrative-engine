package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestManuscriptHistoryService reads one Project's retained Manuscript Versions.
type RestManuscriptHistoryService interface {
	Exec(ctx context.Context, request *core.ManuscriptHistoryRequest) ([]*core.Manuscript, error)
}

// RestManuscriptHistory publishes bounded Manuscript history over HTTP.
type RestManuscriptHistory struct {
	service RestManuscriptHistoryService
	logger  logging.Log
}

// NewRestManuscriptHistory creates a Manuscript history handler.
func NewRestManuscriptHistory(
	service RestManuscriptHistoryService,
	logger logging.Log,
) *RestManuscriptHistory {
	return &RestManuscriptHistory{service: service, logger: logger}
}

// ServeHTTP returns retained Manuscript Versions newest first.
func (handler *RestManuscriptHistory) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ManuscriptHistory")
	defer span.End()

	actor, projectID, err := restProjectIdentity(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restTransportErrors, err)

		return
	}

	manuscripts, err := handler.service.Exec(ctx, &core.ManuscriptHistoryRequest{
		Actor:     *actor,
		ProjectID: projectID,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restServiceErrors, err)

		return
	}

	response := make([]*RestManuscriptVersion, 0, len(manuscripts))
	for _, manuscript := range manuscripts {
		response = append(response, loadRestManuscriptVersion(manuscript))
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, response)
}
