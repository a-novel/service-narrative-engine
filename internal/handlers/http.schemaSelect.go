package handlers

import (
	"context"
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

type SchemaSelectService interface {
	Exec(ctx context.Context, request *services.SchemaSelectRequest) (*services.Schema, error)
}

type SchemaSelectRequest struct {
	ID        *uuid.UUID `schema:"id"`
	ProjectID uuid.UUID  `schema:"projectID"`
	Module    string     `schema:"module"`
}

type SchemaSelect struct {
	service SchemaSelectService
	logger  logging.Log
}

func NewSchemaSelect(service SchemaSelectService, logger logging.Log) *SchemaSelect {
	return &SchemaSelect{service: service, logger: logger}
}

func (handler *SchemaSelect) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.SchemaSelect")
	defer span.End()

	var request SchemaSelectRequest

	err := muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	claims, err := authpkg.MustGetClaimsContext(ctx)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusForbidden}, err)

		return
	}

	res, err := handler.service.Exec(ctx, &services.SchemaSelectRequest{
		ID:        request.ID,
		ProjectID: request.ProjectID,
		Module:    request.Module,
		UserID:    lo.FromPtr(claims.UserID),
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{
			services.ErrInvalidRequest:        http.StatusUnprocessableEntity,
			services.ErrUserDoesNotOwnProject: http.StatusForbidden,
			dao.ErrSchemaSelectNotFound:       http.StatusNotFound,
			dao.ErrProjectSelectNotFound:      http.StatusNotFound,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, loadSchema(res))
}
