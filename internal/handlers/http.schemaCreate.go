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

type SchemaCreateService interface {
	Exec(ctx context.Context, request *services.SchemaCreateRequest) (*services.Schema, error)
}

type SchemaCreateRequest struct {
	ID        uuid.UUID      `json:"id"`
	ProjectID uuid.UUID      `json:"projectID"`
	Module    string         `json:"module"`
	Source    string         `json:"source"`
	Data      map[string]any `json:"data"`
}

type SchemaCreate struct {
	service SchemaCreateService
}

func NewSchemaCreate(service SchemaCreateService) *SchemaCreate {
	return &SchemaCreate{service: service}
}

func (handler *SchemaCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.SchemaCreate")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request SchemaCreateRequest

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

	res, err := handler.service.Exec(ctx, &services.SchemaCreateRequest{
		ID:        request.ID,
		ProjectID: request.ProjectID,
		UserID:    lo.FromPtr(claims.UserID),
		Module:    request.Module,
		Source:    request.Source,
		Data:      request.Data,
	})
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{
			services.ErrInvalidRequest:        http.StatusUnprocessableEntity,
			services.ErrUserDoesNotOwnProject: http.StatusForbidden,
			services.ErrModuleNotInProject:    http.StatusUnprocessableEntity,
			dao.ErrProjectSelectNotFound:      http.StatusNotFound,
			dao.ErrModuleSelectNotFound:       http.StatusNotFound,
			dao.ErrSchemaInsertAlreadyExists:  http.StatusConflict,
		}, err)

		return
	}

	w.WriteHeader(http.StatusCreated)
	httpf.SendJSON(ctx, w, span, loadSchema(res))
}
