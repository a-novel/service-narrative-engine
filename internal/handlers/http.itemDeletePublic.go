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

// ItemDeletePublicService is the service port ItemDeletePublic depends on to delete an item
// and return the deleted record.
type ItemDeletePublicService interface {
	Exec(ctx context.Context, request *core.ItemDeleteRequest) (*core.Item, error)
}

// ItemDeletePublicRequest is the query string accepted by the delete-item endpoint.
type ItemDeletePublicRequest struct {
	ID uuid.UUID `schema:"id"`
}

// ItemDeletePublic is the REST handler that deletes an item by ID.
type ItemDeletePublic struct {
	service ItemDeletePublicService
	logger  logging.Log
}

// NewItemDeletePublic returns a new ItemDeletePublic handler backed by the given service.
func NewItemDeletePublic(service ItemDeletePublicService, logger logging.Log) *ItemDeletePublic {
	return &ItemDeletePublic{service: service, logger: logger}
}

func (handler *ItemDeletePublic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ItemDeletePublic")
	defer span.End()

	actor, err := actorFromContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			ErrAnonymousActor: http.StatusForbidden,
		}, err)

		return
	}

	var request ItemDeletePublicRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	item, err := handler.service.Exec(ctx, &core.ItemDeleteRequest{Actor: *actor, ID: request.ID})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			dao.ErrItemDeleteNotFound: http.StatusNotFound,
			core.ErrInvalidRequest:    http.StatusBadRequest,
		}, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, loadItem(item))
}
