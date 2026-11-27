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

// RestInstanceDeleteService is the core operation [RestInstanceDelete] delegates to.
type RestInstanceDeleteService interface {
	Exec(ctx context.Context, request *core.InstanceDeleteRequest) (*core.InstanceDescriptor, error)
}

// RestInstanceDeleteRequest is the query parameters the delete-instance endpoint accepts.
type RestInstanceDeleteRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
	ID        uuid.UUID `schema:"id"`
}

// RestInstanceDelete is the HTTP handler for the delete-instance endpoint. It responds with the deleted
// instance, without its content.
type RestInstanceDelete struct {
	service RestInstanceDeleteService
	logger  logging.Log
}

func NewRestInstanceDelete(service RestInstanceDeleteService, logger logging.Log) *RestInstanceDelete {
	return &RestInstanceDelete{service: service, logger: logger}
}

func (handler *RestInstanceDelete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.InstanceDelete")
	defer span.End()

	ownerID, err := serviceauthentication.RequestUser(r.Context())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestInstanceDeleteRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	serviceRequest := &core.InstanceDeleteRequest{OwnerID: ownerID, ProjectID: request.ProjectID, ID: request.ID}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, instanceErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadInstanceDescriptor(result))
}
