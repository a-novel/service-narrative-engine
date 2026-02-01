package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	authpkg "github.com/a-novel/service-authentication/v2/pkg"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

type SchemaRewriteService interface {
	Exec(ctx context.Context, request *services.SchemaRewriteRequest) (*services.Schema, error)
}

type SchemaRewriteRequest struct {
	ID   uuid.UUID      `json:"id"`
	Data map[string]any `json:"data"`
}

type SchemaRewrite struct {
	service SchemaRewriteService
	logger  logging.Log
}

func NewSchemaRewrite(service SchemaRewriteService, logger logging.Log) *SchemaRewrite {
	return &SchemaRewrite{service: service, logger: logger}
}

func (handler *SchemaRewrite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.SchemaRewrite")
	defer span.End()

	decoder := json.NewDecoder(r.Body)

	var request SchemaRewriteRequest

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

	res, err := handler.service.Exec(ctx, &services.SchemaRewriteRequest{
		ID:     request.ID,
		UserID: lo.FromPtr(claims.UserID),
		Data:   request.Data,
		Now:    time.Now(),
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
