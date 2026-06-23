package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

type ItemGetPublicService interface {
	Exec(ctx context.Context, request *services.ItemGetRequest) (*services.Item, error)
}

type ItemGetPublicRequest struct {
	ID uuid.UUID `schema:"id"`
}

type ItemGetPublic struct {
	service ItemGetPublicService
	logger  logging.Log
}

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

	item, err := handler.service.Exec(ctx, &services.ItemGetRequest{ID: request.ID})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			dao.ErrItemGetNotFound:     http.StatusNotFound,
			services.ErrInvalidRequest: http.StatusBadRequest,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadItem(item))
}
