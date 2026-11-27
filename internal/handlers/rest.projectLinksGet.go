package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
)

// RestProjectLinksGetService is the core operation [RestProjectLinksGet] delegates to.
type RestProjectLinksGetService interface {
	Exec(ctx context.Context, request *core.ProjectLinksGetRequest) (*core.ProjectLinks, error)
}

// RestProjectLinksGetRequest is the query parameters the get-links endpoint accepts.
type RestProjectLinksGetRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
}

// RestProjectLinksGet is the HTTP handler for the get-links endpoint. It responds with the links of the
// project, and when they last changed.
type RestProjectLinksGet struct {
	service RestProjectLinksGetService
	logger  logging.Log
}

func NewRestProjectLinksGet(service RestProjectLinksGetService, logger logging.Log) *RestProjectLinksGet {
	return &RestProjectLinksGet{service: service, logger: logger}
}

func (handler *RestProjectLinksGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectLinksGet")
	defer span.End()

	ownerID, err := serviceauthentication.RequestUser(r.Context())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectLinksGetRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	serviceRequest := &core.ProjectLinksGetRequest{OwnerID: ownerID, ProjectID: request.ProjectID}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, instanceErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, ProjectLinks{Links: result.Links, UpdatedAt: result.UpdatedAt})
}
