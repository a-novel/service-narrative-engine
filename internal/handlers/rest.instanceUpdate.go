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

// RestInstanceUpdateService is the core operation [RestInstanceUpdate] delegates to.
type RestInstanceUpdateService interface {
	Exec(ctx context.Context, request *core.InstanceUpdateRequest) (*core.InstanceDescriptor, error)
}

// RestInstanceUpdateRequest is the JSON body the update-instance endpoint accepts.
type RestInstanceUpdateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`
	ID        uuid.UUID `json:"id"`

	// Name is the display name the writer gives the instance.
	Name string `json:"name"`
}

// RestInstanceUpdate is the HTTP handler for the update-instance endpoint, which renames an instance. It
// responds with the instance without its content.
type RestInstanceUpdate struct {
	service RestInstanceUpdateService
	logger  logging.Log
}

func NewRestInstanceUpdate(service RestInstanceUpdateService, logger logging.Log) *RestInstanceUpdate {
	return &RestInstanceUpdate{service: service, logger: logger}
}

func (handler *RestInstanceUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.InstanceUpdate")
	defer span.End()

	ownerID, err := serviceauthentication.RequestUser(r.Context())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestInstanceUpdateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	serviceRequest := &core.InstanceUpdateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		ID:        request.ID,
		Name:      request.Name,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, instanceErrors, tagLimit(err))

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadInstanceDescriptor(result))
}
