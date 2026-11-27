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

// RestInstanceLinksListService is the core operation [RestInstanceLinksList] delegates to.
type RestInstanceLinksListService interface {
	Exec(ctx context.Context, request *core.InstanceLinksListRequest) (*core.InstanceLinksPage, error)
}

// RestInstanceLinksListRequest is the query parameters the list-links endpoint accepts.
type RestInstanceLinksListRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`

	// Limit caps the page. Zero takes the default size.
	Limit int `schema:"limit"`
	// Offset is how many instances come before the page, oldest first.
	Offset int `schema:"offset"`
}

// RestInstanceLinksList is the HTTP handler for the list-links endpoint. It responds with a page of the links
// the instances of the project store, oldest instance first, each list with its instance and when it last
// changed, and with the total of the instances.
type RestInstanceLinksList struct {
	service RestInstanceLinksListService
	logger  logging.Log
}

func NewRestInstanceLinksList(service RestInstanceLinksListService, logger logging.Log) *RestInstanceLinksList {
	return &RestInstanceLinksList{service: service, logger: logger}
}

func (handler *RestInstanceLinksList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.InstanceLinksList")
	defer span.End()

	ownerID, err := serviceauthentication.RequestUser(r.Context())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestInstanceLinksListRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.InstanceLinksListRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Limit:     request.Limit,
		Offset:    request.Offset,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, instanceErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadInstanceLinksPage(result))
}
