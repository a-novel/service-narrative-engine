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

// RestInstanceListService is the core operation [RestInstanceList] delegates to.
type RestInstanceListService interface {
	Exec(ctx context.Context, request *core.InstanceListRequest) (*core.InstancePage, error)
}

// RestInstanceListRequest is the query parameters the list-instances endpoint accepts.
type RestInstanceListRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`

	// Limit caps the page. Zero takes the default size.
	Limit int `schema:"limit"`
	// Offset is how many instances come before the page, oldest first.
	Offset int `schema:"offset"`
}

// RestInstanceList is the HTTP handler for the list-instances endpoint. It responds with a page of the
// instances of the project, oldest first, without their content, and their total.
type RestInstanceList struct {
	service RestInstanceListService
	logger  logging.Log
}

func NewRestInstanceList(service RestInstanceListService, logger logging.Log) *RestInstanceList {
	return &RestInstanceList{service: service, logger: logger}
}

func (handler *RestInstanceList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.InstanceList")
	defer span.End()

	ownerID, err := serviceauthentication.RequestUser(r.Context())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestInstanceListRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.InstanceListRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Limit:     request.Limit,
		Offset:    request.Offset,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, instanceErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadInstancePage(result))
}
