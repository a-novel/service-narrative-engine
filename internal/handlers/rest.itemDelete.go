package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-template/internal/core"
)

// RestItemDeleteService is the core operation [RestItemDelete] delegates to.
type RestItemDeleteService interface {
	Exec(ctx context.Context, request *core.ItemDeleteRequest) (*core.Item, error)
}

// itemDeleteErrors maps a refused item deletion to its status.
var itemDeleteErrors = httpf.ErrMap{
	core.ErrItemDeleteNotFound: http.StatusNotFound,
	core.ErrInvalidRequest:     http.StatusBadRequest,
}

// RestItemDeleteRequest carries the query parameters accepted by the delete endpoint.
type RestItemDeleteRequest struct {
	ID uuid.UUID `schema:"id"`
}

// RestItemDelete is the HTTP handler for the delete-item endpoint. It responds
// with the deleted item.
type RestItemDelete struct {
	service RestItemDeleteService
	logger  logging.Log
}

func NewRestItemDelete(service RestItemDeleteService, logger logging.Log) *RestItemDelete {
	return &RestItemDelete{service: service, logger: logger}
}

func (handler *RestItemDelete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ItemDelete")
	defer span.End()

	var request RestItemDeleteRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	item, err := handler.service.Exec(ctx, &core.ItemDeleteRequest{ID: request.ID})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, itemDeleteErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadItem(item))
}
