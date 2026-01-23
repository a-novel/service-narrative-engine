package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-template/internal/services"
)

type ItemListPublicService interface {
	Exec(ctx context.Context, request *services.ItemListRequest) ([]*services.Item, error)
}

type ItemListPublicRequest struct {
	Limit  int `schema:"limit"`
	Offset int `schema:"offset"`
}

type ItemListPublic struct {
	service ItemListPublicService
	logger  logging.Log
}

func NewItemListPublic(service ItemListPublicService, logger logging.Log) *ItemListPublic {
	return &ItemListPublic{service: service, logger: logger}
}

func (handler *ItemListPublic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ItemListPublic")
	defer span.End()

	var request ItemListPublicRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	items, err := handler.service.Exec(ctx, &services.ItemListRequest{
		Limit:  request.Limit,
		Offset: request.Offset,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			services.ErrInvalidRequest: http.StatusBadRequest,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, lo.Map(items, loadItemMap))
}
