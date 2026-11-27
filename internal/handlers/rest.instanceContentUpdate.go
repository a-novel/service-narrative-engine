package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
)

// RestInstanceContentUpdateService is the core operation [RestInstanceContentUpdate] delegates to.
type RestInstanceContentUpdateService interface {
	Exec(ctx context.Context, request *core.InstanceContentUpdateRequest) (*core.InstanceDescriptor, error)
}

// RestInstanceContentUpdateRequest is the JSON body the update-instance-content endpoint accepts.
type RestInstanceContentUpdateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`
	ID        uuid.UUID `json:"id"`

	// Version is the exact version of the plugin the content follows, as a semantic version.
	Version string `json:"version"`
	// Content is the JSON document the plugin edits.
	Content json.RawMessage `json:"content"`

	// UpdatedAt is the updatedAt the instance carried when the client read it, sent back as it was received.
	UpdatedAt time.Time `json:"updatedAt"`
}

// RestInstanceContentUpdate is the HTTP handler for the update-instance-content endpoint. It responds with
// the instance without its content, whose updatedAt the next write names.
type RestInstanceContentUpdate struct {
	service RestInstanceContentUpdateService
	logger  logging.Log
}

func NewRestInstanceContentUpdate(
	service RestInstanceContentUpdateService, logger logging.Log,
) *RestInstanceContentUpdate {
	return &RestInstanceContentUpdate{service: service, logger: logger}
}

func (handler *RestInstanceContentUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.InstanceContentUpdate")
	defer span.End()

	ownerID, err := serviceauthentication.RequestUser(r.Context())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestInstanceContentUpdateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	serviceRequest := &core.InstanceContentUpdateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		ID:        request.ID,
		Version:   request.Version,
		Content:   request.Content,
		UpdatedAt: request.UpdatedAt,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, instanceErrors, tagLimit(tagField(err)))

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadInstanceDescriptor(result))
}
