package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ItemGetPublicService is the service port ItemGetPublic depends on to fetch a single item.
type ItemGetPublicService interface {
	Exec(ctx context.Context, request *core.ItemGetRequest) (*core.Item, error)
}

// ItemGetPublicRequest is the query string accepted by the get-item endpoint.
type ItemGetPublicRequest struct {
	ID uuid.UUID `schema:"id"`
}

// ItemGetPublic is the REST handler that fetches a single item by ID.
type ItemGetPublic struct {
	service ItemGetPublicService
	logger  logging.Log
}

// NewItemGetPublic returns a new ItemGetPublic handler backed by the given service.
func NewItemGetPublic(service ItemGetPublicService, logger logging.Log) *ItemGetPublic {
	return &ItemGetPublic{service: service, logger: logger}
}

func (handler *ItemGetPublic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ItemGetPublic")
	defer span.End()

	var request ItemGetPublicRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	item, err := handler.service.Exec(ctx, &core.ItemGetRequest{ID: request.ID})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			dao.ErrItemGetNotFound: http.StatusNotFound,
			core.ErrInvalidRequest: http.StatusBadRequest,
		}, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, loadItem(item))
}
