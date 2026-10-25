package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-template/internal/core"
)

// RestItemCreateService is the core operation [RestItemCreate] delegates to.
type RestItemCreateService interface {
	Exec(ctx context.Context, request *core.ItemCreateRequest) (*core.Item, error)
}

// itemCreateErrors maps a refused item creation to its status.
var itemCreateErrors = httpf.ErrMap{
	core.ErrInvalidRequest: http.StatusBadRequest,
}

// RestItemCreateRequest is the JSON body accepted by the create endpoint.
type RestItemCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// RestItemCreate is the HTTP handler for the create-item endpoint. It responds
// 201 with the created item.
type RestItemCreate struct {
	service RestItemCreateService
	logger  logging.Log
}

func NewRestItemCreate(service RestItemCreateService, logger logging.Log) *RestItemCreate {
	return &RestItemCreate{service: service, logger: logger}
}

func (handler *RestItemCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ItemCreate")
	defer span.End()

	var request RestItemCreateRequest

	err := httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.ItemCreateRequest{
		Name:        request.Name,
		Description: request.Description,
	}

	item, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, itemCreateErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusCreated, loadItem(item))
}
