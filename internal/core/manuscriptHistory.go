package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ManuscriptHistoryDao retrieves retained Manuscript Versions.
type ManuscriptHistoryDao interface {
	Exec(ctx context.Context, request *dao.ManuscriptListRequest) ([]*dao.Manuscript, error)
}

// ManuscriptHistoryRequest identifies an owned Project's Manuscript history.
type ManuscriptHistoryRequest struct {
	Actor     Actor     `validate:"actor"`
	ProjectID uuid.UUID `validate:"required"`
}

// ManuscriptHistory reads the bounded, newest-first Manuscript history.
type ManuscriptHistory struct {
	projectAccess ProjectAccessService
	dao           ManuscriptHistoryDao
}

// NewManuscriptHistory creates an owner-scoped Manuscript history service.
func NewManuscriptHistory(
	projectAccess ProjectAccessService,
	historyDao ManuscriptHistoryDao,
) *ManuscriptHistory {
	return &ManuscriptHistory{projectAccess: projectAccess, dao: historyDao}
}

// Exec returns all retained Manuscript Versions after checking Project ownership.
func (service *ManuscriptHistory) Exec(
	ctx context.Context,
	request *ManuscriptHistoryRequest,
) ([]*Manuscript, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ManuscriptHistory")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.Actor.UserID.String()),
	)

	_, err = service.projectAccess.Exec(ctx, &ProjectAccessRequest{
		Actor:     request.Actor,
		ProjectID: request.ProjectID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("access Project: %w", err))
	}

	entities, err := service.dao.Exec(ctx, &dao.ManuscriptListRequest{ProjectID: request.ProjectID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list Manuscripts: %w", err))
	}

	manuscripts := make([]*Manuscript, 0, len(entities))
	for _, entity := range entities {
		manuscripts = append(manuscripts, manuscriptFromDao(entity))
	}

	return otel.ReportSuccess(span, manuscripts), nil
}
