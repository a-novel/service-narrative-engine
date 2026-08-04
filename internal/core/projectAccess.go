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

var errProjectAccessIdeaMissing = errors.New("Idea selection returned no entity")

// IdeaSelectDao retrieves one owner-scoped Idea.
type IdeaSelectDao interface {
	Exec(ctx context.Context, request *dao.IdeaSelectRequest) (*dao.Idea, error)
}

// ProjectAccessService resolves an Idea only when the actor owns its project.
type ProjectAccessService interface {
	Exec(ctx context.Context, request *ProjectAccessRequest) (*dao.Idea, error)
}

// ProjectAccessRequest identifies an actor and the project rooted at an Idea.
type ProjectAccessRequest struct {
	Actor  Actor     `validate:"actor"`
	IdeaID uuid.UUID `validate:"required"`
}

// ProjectAccess centralizes the current project ownership rule.
type ProjectAccess struct {
	ideaDao IdeaSelectDao
}

// NewProjectAccess creates a project access service.
func NewProjectAccess(ideaDao IdeaSelectDao) *ProjectAccess {
	return &ProjectAccess{ideaDao: ideaDao}
}

// Exec returns the project Idea when the actor is its owner.
func (service *ProjectAccess) Exec(
	ctx context.Context,
	request *ProjectAccessRequest,
) (*dao.Idea, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectAccess")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	span.SetAttributes(
		attribute.String("idea.id", request.IdeaID.String()),
		attribute.String("idea.owner_id", request.Actor.UserID.String()),
	)

	idea, err := service.ideaDao.Exec(ctx, &dao.IdeaSelectRequest{
		ID:      request.IdeaID,
		OwnerID: request.Actor.UserID,
	})
	if errors.Is(err, dao.ErrIdeaSelectNotFound) {
		err = errors.Join(err, ErrIdeaNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select Idea: %w", err))
	}

	if idea == nil {
		return nil, otel.ReportError(span, fmt.Errorf("select Idea: %w", errProjectAccessIdeaMissing))
	}

	return otel.ReportSuccess(span, idea), nil
}
