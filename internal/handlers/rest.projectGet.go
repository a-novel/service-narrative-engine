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

// RestProjectGetService is the core operation [RestProjectGet] delegates to.
type RestProjectGetService interface {
	Exec(ctx context.Context, request *core.ProjectGetRequest) (*core.Project, error)
}

// RestProjectGetRequest is the query parameters the get-project endpoint accepts.
type RestProjectGetRequest struct {
	ID uuid.UUID `schema:"id"`
}

// RestProjectGet is the HTTP handler for the get-project endpoint. It responds with the project. Its
// scenes, layers and cells come page by page from their list endpoints.
type RestProjectGet struct {
	service RestProjectGetService
	logger  logging.Log
}

func NewRestProjectGet(service RestProjectGetService, logger logging.Log) *RestProjectGet {
	return &RestProjectGet{service: service, logger: logger}
}

func (handler *RestProjectGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectGet")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectGetRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.ProjectGetRequest{
		OwnerID: ownerID,
		ID:      request.ID,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadProject(result))
}
