package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ItemUpdatePublicService is the service port ItemUpdatePublic depends on to update an item
// and return the updated record.
type ItemUpdatePublicService interface {
	Exec(ctx context.Context, request *core.ItemUpdateRequest) (*core.Item, error)
}

// ItemUpdatePublicRequest is the JSON body accepted by the update-item endpoint.
type ItemUpdatePublicRequest struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// ItemUpdatePublic is the REST handler that updates an existing item.
type ItemUpdatePublic struct {
	service ItemUpdatePublicService
	logger  logging.Log
}

// NewItemUpdatePublic returns a new ItemUpdatePublic handler backed by the given service.
func NewItemUpdatePublic(service ItemUpdatePublicService, logger logging.Log) *ItemUpdatePublic {
	return &ItemUpdatePublic{service: service, logger: logger}
}

func (handler *ItemUpdatePublic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ItemUpdatePublic")
	defer span.End()

	actor, err := actorFromContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			ErrAnonymousActor: http.StatusForbidden,
		}, err)

		return
	}

	decoder := json.NewDecoder(r.Body)

	var request ItemUpdatePublicRequest

	err = decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	item, err := handler.service.Exec(ctx, &core.ItemUpdateRequest{
		Actor:       *actor,
		ID:          request.ID,
		Name:        request.Name,
		Description: request.Description,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			dao.ErrItemUpdateNotFound: http.StatusNotFound,
			core.ErrInvalidRequest:    http.StatusBadRequest,
		}, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, loadItem(item))
}
