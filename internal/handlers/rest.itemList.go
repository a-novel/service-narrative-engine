package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestItemListService is the core operation [RestItemList] delegates to.
type RestItemListService interface {
	Exec(ctx context.Context, request *core.ItemListRequest) ([]*core.Item, error)
}

// itemListErrors maps a refused page request to its status.
var itemListErrors = httpf.ErrMap{
	core.ErrInvalidRequest: http.StatusBadRequest,
}

// RestItemListRequest carries the pagination query parameters accepted by the
// list endpoint.
type RestItemListRequest struct {
	Limit int       `schema:"limit"`
	After uuid.UUID `schema:"after"`
}

// RestItemList is the HTTP handler for the list-items endpoint.
type RestItemList struct {
	service RestItemListService
	logger  logging.Log
}

func NewRestItemList(service RestItemListService, logger logging.Log) *RestItemList {
	return &RestItemList{service: service, logger: logger}
}

func (handler *RestItemList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ItemList")
	defer span.End()

	var request RestItemListRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.ItemListRequest{
		Limit: request.Limit,
		After: request.After,
	}

	items, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, itemListErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, lo.Map(items, loadItemMap))
}
