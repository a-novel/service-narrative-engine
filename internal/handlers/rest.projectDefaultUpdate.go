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

// RestProjectDefaultUpdateService is the core operation [RestProjectDefaultUpdate] delegates to.
type RestProjectDefaultUpdateService interface {
	Exec(ctx context.Context, request *core.ProjectDefaultUpdateRequest) (*core.Project, error)
}

// RestProjectDefaultUpdateRequest is the JSON body the update-project-default endpoint accepts.
type RestProjectDefaultUpdateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`
	// InstanceID is the instance the project opens on from now on.
	InstanceID uuid.UUID `json:"instanceID"`
}

// RestProjectDefaultUpdate is the HTTP handler for the update-project-default endpoint, which chooses the
// default instance of a project. It responds with the project.
type RestProjectDefaultUpdate struct {
	service RestProjectDefaultUpdateService
	logger  logging.Log
}

func NewRestProjectDefaultUpdate(
	service RestProjectDefaultUpdateService, logger logging.Log,
) *RestProjectDefaultUpdate {
	return &RestProjectDefaultUpdate{service: service, logger: logger}
}

func (handler *RestProjectDefaultUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectDefaultUpdate")
	defer span.End()

	ownerID, err := serviceauthentication.RequestUser(r.Context())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectDefaultUpdateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	serviceRequest := &core.ProjectDefaultUpdateRequest{
		OwnerID:    ownerID,
		ProjectID:  request.ProjectID,
		InstanceID: request.InstanceID,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, instanceErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadProject(result))
}
