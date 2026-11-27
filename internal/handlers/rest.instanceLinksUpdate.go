package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
)

// RestInstanceLinksUpdateService is the core operation [RestInstanceLinksUpdate] delegates to.
type RestInstanceLinksUpdateService interface {
	Exec(ctx context.Context, request *core.InstanceLinksUpdateRequest) (*core.WrittenInstanceLinks, error)
}

// RestInstanceLinksUpdateRequest is the JSON body the update-instance-links endpoint accepts.
type RestInstanceLinksUpdateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`
	ID        uuid.UUID `json:"id"`

	// Links replace the links the instance stores. An empty list clears them, and an omitted one is refused.
	Links []Link `json:"links"`

	// UpdatedAt is the updatedAt the links carried when the client read them, sent back as it was received:
	// null or omitted for an instance that never wrote its links.
	UpdatedAt *time.Time `json:"updatedAt"`
}

// RestInstanceLinksUpdate is the HTTP handler for the update-instance-links endpoint. It responds with the
// updatedAt of the written links, which the next write names.
type RestInstanceLinksUpdate struct {
	service RestInstanceLinksUpdateService
	logger  logging.Log
}

func NewRestInstanceLinksUpdate(
	service RestInstanceLinksUpdateService, logger logging.Log,
) *RestInstanceLinksUpdate {
	return &RestInstanceLinksUpdate{service: service, logger: logger}
}

func (handler *RestInstanceLinksUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.InstanceLinksUpdate")
	defer span.End()

	ownerID, err := serviceauthentication.RequestUser(r.Context())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestInstanceLinksUpdateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	serviceRequest := &core.InstanceLinksUpdateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		ID:        request.ID,
		UpdatedAt: request.UpdatedAt,
	}

	// A list the body omits stays nil, so the service refuses it.
	if request.Links != nil {
		serviceRequest.Links = lo.Map(request.Links, func(link Link, _ int) core.Link { return loadLink(link) })
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, instanceErrors, tagLimit(tagField(err)))

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, WrittenInstanceLinks{UpdatedAt: result.UpdatedAt})
}
