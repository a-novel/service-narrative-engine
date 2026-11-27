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

// RestInstanceGetService is the core operation [RestInstanceGet] delegates to.
type RestInstanceGetService interface {
	Exec(ctx context.Context, request *core.InstanceGetRequest) (*core.Instance, error)
}

// RestInstanceGetRequest is the query parameters the get-instance endpoint accepts.
type RestInstanceGetRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
	ID        uuid.UUID `schema:"id"`
}

// RestInstanceGet is the HTTP handler for the get-instance endpoint. It responds with the instance and its
// content.
type RestInstanceGet struct {
	service RestInstanceGetService
	logger  logging.Log
}

func NewRestInstanceGet(service RestInstanceGetService, logger logging.Log) *RestInstanceGet {
	return &RestInstanceGet{service: service, logger: logger}
}

func (handler *RestInstanceGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.InstanceGet")
	defer span.End()

	ownerID, err := serviceauthentication.RequestUser(r.Context())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestInstanceGetRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	serviceRequest := &core.InstanceGetRequest{OwnerID: ownerID, ProjectID: request.ProjectID, ID: request.ID}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, instanceErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, Instance{
		InstanceDescriptor: loadInstanceDescriptor(&result.InstanceDescriptor),
		Content:            result.Content,
	})
}
