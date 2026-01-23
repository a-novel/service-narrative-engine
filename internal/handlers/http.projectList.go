package handlers

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	authpkg "github.com/a-novel/service-authentication/v2/pkg"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/services"
)

type ProjectListService interface {
	Exec(ctx context.Context, request *services.ProjectListRequest) ([]*services.Project, error)
}

type ProjectListRequest struct {
	Limit  int `schema:"limit"`
	Offset int `schema:"offset"`
}

type ProjectList struct {
	service ProjectListService
}

func NewProjectList(service ProjectListService) *ProjectList {
	return &ProjectList{service: service}
}

func (handler *ProjectList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ProjectList")
	defer span.End()

	var request ProjectListRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	claims, err := authpkg.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusForbidden}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.ProjectListRequest{
		UserID: lo.FromPtr(claims.UserID),
		Limit:  request.Limit,
		Offset: request.Offset,
	})
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{
			services.ErrInvalidRequest: http.StatusUnprocessableEntity,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, lo.Map(res, loadProjectMap))
}
