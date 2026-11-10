package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestProjectLogoDeleteService is the core operation [RestProjectLogoDelete] delegates to.
type RestProjectLogoDeleteService interface {
	Exec(ctx context.Context, request *core.ProjectLogoDeleteRequest) (*core.Project, error)
}

// RestProjectLogoDeleteRequest is the query parameters the delete-logo endpoint accepts.
type RestProjectLogoDeleteRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
}

// RestProjectLogoDelete is the HTTP handler for the delete-logo endpoint. It responds with the project,
// without its logo.
type RestProjectLogoDelete struct {
	service RestProjectLogoDeleteService
	logger  logging.Log
}

func NewRestProjectLogoDelete(
	service RestProjectLogoDeleteService, logger logging.Log,
) *RestProjectLogoDelete {
	return &RestProjectLogoDelete{service: service, logger: logger}
}

func (handler *RestProjectLogoDelete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectLogoDelete")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectLogoDeleteRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.ProjectLogoDeleteRequest{
		OwnerID: ownerID, ProjectID: request.ProjectID,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, logoErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadProject(result))
}
