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

// RestProjectFieldsDriftService is the core operation [RestProjectFieldsDrift] delegates to.
type RestProjectFieldsDriftService interface {
	Exec(ctx context.Context, request *core.ProjectFieldsDriftRequest) (*core.ProjectFieldsDriftResult, error)
}

// RestProjectFieldsDriftRequest is the JSON body the fields drift endpoint accepts.
type RestProjectFieldsDriftRequest struct {
	// Projects maps the ID of each project the client holds to its updatedAt, compared to the millisecond.
	Projects map[string]string `json:"projects"`
}

// RestProjectFieldsDriftResult is the REST representation of the drift between the projects a client holds and
// the caller's.
type RestProjectFieldsDriftResult struct {
	// Projects are the caller's projects the request leaves out, or names with another updatedAt.
	Projects []ProjectFields `json:"projects"`
	// Removed are the IDs of the request that name no project of the caller.
	Removed []uuid.UUID `json:"removed"`
}

// RestProjectFieldsDrift is the HTTP handler for the fields drift endpoint. It responds with the caller's projects
// whose fields drifted from those the client holds, and the held IDs that name no project of the caller.
type RestProjectFieldsDrift struct {
	service RestProjectFieldsDriftService
	logger  logging.Log
}

func NewRestProjectFieldsDrift(service RestProjectFieldsDriftService, logger logging.Log) *RestProjectFieldsDrift {
	return &RestProjectFieldsDrift{service: service, logger: logger}
}

func (handler *RestProjectFieldsDrift) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectFieldsDrift")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectFieldsDriftRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.ProjectFieldsDriftRequest{
		OwnerID:  ownerID,
		Projects: request.Projects,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, RestProjectFieldsDriftResult{
		Projects: lo.Map(result.Projects, func(project *core.Project, _ int) ProjectFields {
			return loadProjectFields(project)
		}),
		Removed: result.Removed,
	})
}
