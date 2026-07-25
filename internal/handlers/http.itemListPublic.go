package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// ItemListPublicService is the service port ItemListPublic depends on to list items.
type ItemListPublicService interface {
	Exec(ctx context.Context, request *core.ItemListRequest) ([]*core.Item, error)
}

// ItemListPublicRequest is the query string accepted by the list-items endpoint. Limit and
// Offset paginate the result set.
type ItemListPublicRequest struct {
	Limit  int `schema:"limit"`
	Offset int `schema:"offset"`
}

// ItemListPublic is the REST handler that lists items page by page.
type ItemListPublic struct {
	service ItemListPublicService
	logger  logging.Log
}

// NewItemListPublic returns a new ItemListPublic handler backed by the given service.
func NewItemListPublic(service ItemListPublicService, logger logging.Log) *ItemListPublic {
	return &ItemListPublic{service: service, logger: logger}
}

func (handler *ItemListPublic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ItemListPublic")
	defer span.End()

	actor, err := actorFromContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			ErrAnonymousActor: http.StatusForbidden,
		}, err)

		return
	}

	var request ItemListPublicRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	items, err := handler.service.Exec(ctx, &core.ItemListRequest{
		Actor:  *actor,
		Limit:  request.Limit,
		Offset: request.Offset,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			core.ErrInvalidRequest: http.StatusBadRequest,
		}, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, lo.Map(items, loadItemMap))
}
