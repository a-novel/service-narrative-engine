package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/samber/lo"

	authpkg "github.com/a-novel/service-authentication/v2/pkg"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

type ProjectUpdateService interface {
	Exec(ctx context.Context, request *services.ProjectUpdateRequest) (*services.Project, error)
}

type ProjectUpdateRequest struct {
	ID       uuid.UUID `json:"id"`
	Workflow []string  `json:"workflow"`
	Title    string    `json:"title"`
}

type ProjectUpdate struct {
	service ProjectUpdateService
	logger  logging.Log
}

func NewProjectUpdate(service ProjectUpdateService, logger logging.Log) *ProjectUpdate {
	return &ProjectUpdate{service: service, logger: logger}
}

func (handler *ProjectUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ProjectUpdate")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request ProjectUpdateRequest

	err := decoder.Decode(&request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	claims, err := authpkg.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusForbidden}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.ProjectUpdateRequest{
		ID:       request.ID,
		UserID:   lo.FromPtr(claims.UserID),
		Workflow: request.Workflow,
		Title:    request.Title,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			services.ErrInvalidRequest:         http.StatusUnprocessableEntity,
			services.ErrUserDoesNotOwnProject:  http.StatusForbidden,
			services.ErrForbiddenModuleUpgrade: http.StatusUnprocessableEntity,
			dao.ErrProjectSelectNotFound:       http.StatusNotFound,
			dao.ErrModuleSelectNotFound:        http.StatusNotFound,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadProject(res))
}
