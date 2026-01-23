package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	authpkg "github.com/a-novel/service-authentication/v2/pkg"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

type SchemaListVersionsService interface {
	Exec(ctx context.Context, request *services.SchemaListVersionsRequest) ([]*services.SchemaVersion, error)
}

type SchemaListVersionsRequest struct {
	ProjectID       uuid.UUID `schema:"projectID"`
	ModuleID        string    `schema:"moduleID"`
	ModuleNamespace string    `schema:"moduleNamespace"`
	Limit           int       `schema:"limit"`
	Offset          int       `schema:"offset"`
}

type SchemaVersionResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

type SchemaListVersions struct {
	service SchemaListVersionsService
}

func NewSchemaListVersions(service SchemaListVersionsService) *SchemaListVersions {
	return &SchemaListVersions{service: service}
}

func (handler *SchemaListVersions) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "handler.SchemaListVersions")
	defer span.End()

	var request SchemaListVersionsRequest

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

	res, err := handler.service.Exec(ctx, &services.SchemaListVersionsRequest{
		ProjectID:       request.ProjectID,
		UserID:          lo.FromPtr(claims.UserID),
		ModuleID:        request.ModuleID,
		ModuleNamespace: request.ModuleNamespace,
		Limit:           request.Limit,
		Offset:          request.Offset,
	})
	if err != nil {
		httpf.HandleError(ctx, w, span, httpf.ErrMap{
			services.ErrInvalidRequest:        http.StatusUnprocessableEntity,
			services.ErrUserDoesNotOwnProject: http.StatusForbidden,
			dao.ErrProjectSelectNotFound:      http.StatusNotFound,
		}, err)

		return
	}

	httpf.SendJSON(ctx, w, span, lo.Map(res, loadSchemaVersionsMap))
}
