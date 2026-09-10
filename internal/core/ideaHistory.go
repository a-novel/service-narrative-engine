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

// IdeaHistoryDao retrieves retained Idea Versions.
type IdeaHistoryDao interface {
	Exec(ctx context.Context, request *dao.IdeaVersionListRequest) ([]*dao.IdeaVersion, error)
}

// IdeaHistoryRequest identifies an owned Project's Idea history.
type IdeaHistoryRequest struct {
	Actor     Actor     `validate:"actor"`
	ProjectID uuid.UUID `validate:"required"`
}

// IdeaHistory reads the bounded, newest-first Idea history.
type IdeaHistory struct {
	projectAccess ProjectAccessService
	dao           IdeaHistoryDao
}

// NewIdeaHistory creates an owner-scoped Idea history service.
func NewIdeaHistory(projectAccess ProjectAccessService, historyDao IdeaHistoryDao) *IdeaHistory {
	return &IdeaHistory{projectAccess: projectAccess, dao: historyDao}
}

// Exec returns all retained Idea Versions after checking Project ownership.
func (service *IdeaHistory) Exec(
	ctx context.Context,
	request *IdeaHistoryRequest,
) ([]*Idea, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.IdeaHistory")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.Actor.UserID.String()),
	)

	project, err := service.projectAccess.Exec(ctx, &ProjectAccessRequest{
		Actor:     request.Actor,
		ProjectID: request.ProjectID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("access Project: %w", err))
	}

	entities, err := service.dao.Exec(ctx, &dao.IdeaVersionListRequest{ProjectID: request.ProjectID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list Idea Versions: %w", err))
	}

	ideas := make([]*Idea, 0, len(entities))
	for _, entity := range entities {
		ideas = append(ideas, &Idea{
			ProjectID:        project.ID,
			VersionID:        entity.ID,
			OwnerID:          project.OwnerID,
			Seed:             entity.Seed,
			Genre:            entity.Genre,
			Title:            entity.Title,
			ProjectCreatedAt: project.CreatedAt,
			CreatedAt:        entity.CreatedAt,
		})
	}

	return otel.ReportSuccess(span, ideas), nil
}
