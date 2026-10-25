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

// RestProjectDeleteService is the core operation [RestProjectDelete] delegates to.
type RestProjectDeleteService interface {
	Exec(ctx context.Context, request *core.ProjectDeleteRequest) (*core.Project, error)
}

// RestProjectDeleteRequest is the query parameters the delete-project endpoint accepts.
type RestProjectDeleteRequest struct {
	ID       uuid.UUID `schema:"id"`
	Revision int       `schema:"revision"`
}

// RestProjectDelete is the HTTP handler for the delete-project endpoint. It responds with the deleted
// project.
type RestProjectDelete struct {
	service RestProjectDeleteService
	logger  logging.Log
}

func NewRestProjectDelete(service RestProjectDeleteService, logger logging.Log) *RestProjectDelete {
	return &RestProjectDelete{service: service, logger: logger}
}

func (handler *RestProjectDelete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectDelete")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectDeleteRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.ProjectDeleteRequest{
		OwnerID:  ownerID,
		ID:       request.ID,
		Revision: request.Revision,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadProject(result))
}
