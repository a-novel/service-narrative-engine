package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestProjectGetService reads one owned Project snapshot.
type RestProjectGetService interface {
	Exec(ctx context.Context, request *core.ProjectGetRequest) (*core.ProjectSnapshot, error)
}

// RestProjectGet publishes current Project snapshots over HTTP.
type RestProjectGet struct {
	service RestProjectGetService
	logger  logging.Log
}

// NewRestProjectGet creates a Project snapshot handler.
func NewRestProjectGet(service RestProjectGetService, logger logging.Log) *RestProjectGet {
	return &RestProjectGet{service: service, logger: logger}
}

// ServeHTTP returns the latest content for an owned Project.
func (handler *RestProjectGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectGet")
	defer span.End()

	actor, projectID, err := restProjectIdentity(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restTransportErrors, err)

		return
	}

	project, err := handler.service.Exec(ctx, &core.ProjectGetRequest{
		Actor:     *actor,
		ProjectID: projectID,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restServiceErrors, err)

		return
	}

	stepValues := make([]*RestStepValueVersion, 0, len(project.StepValues))
	for _, value := range project.StepValues {
		stepValues = append(stepValues, loadRestStepValueVersion(value))
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, &RestProject{
		ID:         project.ID,
		CreatedAt:  project.CreatedAt,
		Idea:       loadRestIdeaVersion(project.Idea),
		StepValues: stepValues,
		Manuscript: loadRestManuscriptVersion(project.Manuscript),
	})
}
