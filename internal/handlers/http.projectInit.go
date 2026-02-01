package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/samber/lo"

	authpkg "github.com/a-novel/service-authentication/v2/pkg"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

type ProjectInitService interface {
	Exec(ctx context.Context, request *services.ProjectInitRequest) (*services.Project, error)
}

type ProjectInitRequest struct {
	Lang     string   `json:"lang"`
	Title    string   `json:"title"`
	Workflow []string `json:"workflow"`
}

type ProjectInit struct {
	service ProjectInitService
	logger  logging.Log
}

func NewProjectInit(service ProjectInitService, logger logging.Log) *ProjectInit {
	return &ProjectInit{service: service, logger: logger}
}

func (handler *ProjectInit) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.ProjectInit")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request ProjectInitRequest

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

	res, err := handler.service.Exec(ctx, &services.ProjectInitRequest{
		Owner:    lo.FromPtr(claims.UserID),
		Lang:     request.Lang,
		Title:    request.Title,
		Workflow: request.Workflow,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			services.ErrInvalidRequest:        http.StatusUnprocessableEntity,
			dao.ErrModuleSelectNotFound:       http.StatusNotFound,
			dao.ErrProjectInsertAlreadyExists: http.StatusConflict,
		}, err)

		return
	}

	w.WriteHeader(http.StatusCreated)
	httpf.SendJSON(ctx, w, span, loadProject(res))
}
