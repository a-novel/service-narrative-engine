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

type ItemUpdatePublicService interface {
	Exec(ctx context.Context, request *core.ItemUpdateRequest) (*core.Item, error)
}

type ItemUpdatePublicRequest struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

type ItemUpdatePublic struct {
	service ItemUpdatePublicService
	logger  logging.Log
}

func NewItemUpdatePublic(service ItemUpdatePublicService, logger logging.Log) *ItemUpdatePublic {
	return &ItemUpdatePublic{service: service, logger: logger}
}

func (handler *ItemUpdatePublic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ItemUpdatePublic")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request ItemUpdatePublicRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	item, err := handler.service.Exec(ctx, &core.ItemUpdateRequest{
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

	httpf.SendJSON(ctx, w, span, loadItem(item))
}
