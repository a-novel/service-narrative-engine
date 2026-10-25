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

// RestItemGetService is the core operation [RestItemGet] delegates to.
type RestItemGetService interface {
	Exec(ctx context.Context, request *core.ItemGetRequest) (*core.Item, error)
}

// itemGetErrors maps a failed item lookup to its status.
var itemGetErrors = httpf.ErrMap{
	core.ErrItemGetNotFound: http.StatusNotFound,
	core.ErrInvalidRequest:  http.StatusBadRequest,
}

// RestItemGetRequest carries the query parameters accepted by the get endpoint.
type RestItemGetRequest struct {
	ID uuid.UUID `schema:"id"`
}

// RestItemGet is the HTTP handler for the get-item endpoint.
type RestItemGet struct {
	service RestItemGetService
	logger  logging.Log
}

func NewRestItemGet(service RestItemGetService, logger logging.Log) *RestItemGet {
	return &RestItemGet{service: service, logger: logger}
}

func (handler *RestItemGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ItemGet")
	defer span.End()

	var request RestItemGetRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	item, err := handler.service.Exec(ctx, &core.ItemGetRequest{ID: request.ID})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, itemGetErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadItem(item))
}
