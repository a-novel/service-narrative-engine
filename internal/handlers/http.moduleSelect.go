package handlers

import (
	"context"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

type ModuleSelectService interface {
	Exec(ctx context.Context, request *services.ModuleSelectRequest) (*services.Module, error)
}

type ModuleSelectRequest struct {
	Module string `schema:"module"`
}

type ModuleSelect struct {
	service ModuleSelectService
}

func NewModuleSelect(service ModuleSelectService) *ModuleSelect {
	return &ModuleSelect{service: service}
}

func (handler *ModuleSelect) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ModuleSelect")
	defer span.End()

	var request ModuleSelectRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.ModuleSelectRequest{
		Module: request.Module,
	})
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{
			services.ErrInvalidRequest:  http.StatusUnprocessableEntity,
			dao.ErrModuleSelectNotFound: http.StatusNotFound,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadModule(res))
}
