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

// RestProjectLinksUpdateService is the core operation [RestProjectLinksUpdate] delegates to.
type RestProjectLinksUpdateService interface {
	Exec(ctx context.Context, request *core.ProjectLinksUpdateRequest) (*core.WrittenProjectLinks, error)
}

// RestProjectLinksUpdateRequest is the JSON body the update-links endpoint accepts.
type RestProjectLinksUpdateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Links replace the links of the project. An empty list clears them, and an omitted one is refused.
	Links []Link `json:"links"`

	// UpdatedAt is the updatedAt the links carried when the client read them, sent back as it was received:
	// null or omitted when the project had never written its links.
	UpdatedAt *time.Time `json:"updatedAt"`
}

// RestProjectLinksUpdate is the HTTP handler for the update-links endpoint. It responds with the updatedAt of
// the written links, which the next write names.
type RestProjectLinksUpdate struct {
	service RestProjectLinksUpdateService
	logger  logging.Log
}

func NewRestProjectLinksUpdate(service RestProjectLinksUpdateService, logger logging.Log) *RestProjectLinksUpdate {
	return &RestProjectLinksUpdate{service: service, logger: logger}
}

func (handler *RestProjectLinksUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectLinksUpdate")
	defer span.End()

	ownerID, err := serviceauthentication.RequestUser(r.Context())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectLinksUpdateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	serviceRequest := &core.ProjectLinksUpdateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
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

	httpf.SendJSONStatus(w, span, http.StatusOK, WrittenProjectLinks{UpdatedAt: result.UpdatedAt})
}
