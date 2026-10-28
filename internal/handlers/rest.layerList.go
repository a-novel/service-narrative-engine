package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestLayerListService is the core operation [RestLayerList] delegates to.
type RestLayerListService interface {
	Exec(ctx context.Context, request *core.LayerListRequest) (*core.LayerPage, error)
}

// RestLayerListRequest is the query parameters the list-layers endpoint accepts.
type RestLayerListRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`

	// Limit caps the page. Zero takes the default size.
	Limit int `schema:"limit"`
	// Offset is how many layers come before the page, in order.
	Offset int `schema:"offset"`
}

// RestLayerList is the HTTP handler for the list-layers endpoint. It responds with a page of a
// project's layers, in order, and their total.
type RestLayerList struct {
	service RestLayerListService
	logger  logging.Log
}

func NewRestLayerList(service RestLayerListService, logger logging.Log) *RestLayerList {
	return &RestLayerList{service: service, logger: logger}
}

func (handler *RestLayerList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.LayerList")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestLayerListRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.LayerListRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Limit:     request.Limit,
		Offset:    request.Offset,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadLayerPage(result))
}
