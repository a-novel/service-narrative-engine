package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/samber/lo"

	authpkg "github.com/a-novel/service-authentication/v2/pkg"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

type ProjectDeleteService interface {
	Exec(ctx context.Context, request *services.ProjectDeleteRequest) (*services.Project, error)
}

type ProjectDeleteRequest struct {
	ID uuid.UUID `json:"id"`
}

type ProjectDelete struct {
	service ProjectDeleteService
}

func NewProjectDelete(service ProjectDeleteService) *ProjectDelete {
	return &ProjectDelete{service: service}
}

func (handler *ProjectDelete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ProjectDelete")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request ProjectDeleteRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	claims, err := authpkg.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{nil: http.StatusForbidden}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.ProjectDeleteRequest{
		ID:     request.ID,
		UserID: lo.FromPtr(claims.UserID),
	})
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{
			services.ErrInvalidRequest:        http.StatusUnprocessableEntity,
			services.ErrUserDoesNotOwnProject: http.StatusForbidden,
			dao.ErrProjectSelectNotFound:      http.StatusNotFound,
			dao.ErrProjectDeleteNotFound:      http.StatusNotFound,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadProject(res))
}
