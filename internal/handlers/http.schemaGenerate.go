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

type SchemaGenerateService interface {
	Exec(ctx context.Context, request *services.SchemaGenerateRequest) (*services.Schema, error)
}

type SchemaGenerateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`
	Module    string    `json:"module"`
	Lang      string    `json:"lang"`
}

type SchemaGenerate struct {
	service SchemaGenerateService
	logger  logging.Log
}

func NewSchemaGenerate(service SchemaGenerateService, logger logging.Log) *SchemaGenerate {
	return &SchemaGenerate{service: service, logger: logger}
}

func (handler *SchemaGenerate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.SchemaGenerate")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request SchemaGenerateRequest

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

	res, err := handler.service.Exec(ctx, &services.SchemaGenerateRequest{
		ProjectID: request.ProjectID,
		UserID:    lo.FromPtr(claims.UserID),
		Module:    request.Module,
		Lang:      request.Lang,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
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
