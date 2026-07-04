package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// ItemCreatePublicService is the service port ItemCreatePublic depends on to create an item.
type ItemCreatePublicService interface {
	Exec(ctx context.Context, request *core.ItemCreateRequest) (*core.Item, error)
}

// ItemCreatePublicRequest is the JSON body accepted by the create-item endpoint.
type ItemCreatePublicRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ItemCreatePublic is the REST handler that creates a new item.
type ItemCreatePublic struct {
	service ItemCreatePublicService
	logger  logging.Log
}

// NewItemCreatePublic returns a new ItemCreatePublic handler backed by the given service.
func NewItemCreatePublic(service ItemCreatePublicService, logger logging.Log) *ItemCreatePublic {
	return &ItemCreatePublic{service: service, logger: logger}
}

func (handler *ItemCreatePublic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ItemCreatePublic")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request ItemCreatePublicRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	item, err := handler.service.Exec(ctx, &core.ItemCreateRequest{
		Name:        request.Name,
		Description: request.Description,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			core.ErrInvalidRequest: http.StatusBadRequest,
		}, err)

		return
	}

	w.WriteHeader(http.StatusCreated)
	httpf.SendJSON(ctx, w, span, loadItem(item))
}
