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

// ProjectDeleteDao removes the project.
type ProjectDeleteDao interface {
	Exec(ctx context.Context, request *dao.ProjectDeleteRequest) (*dao.Project, error)
}

// ProjectDeleteRequest names the project to delete.
type ProjectDeleteRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`
}

// ProjectDelete deletes a project with its scenes, layers and cells.
type ProjectDelete struct {
	dao ProjectDeleteDao
}

func NewProjectDelete(dao ProjectDeleteDao) *ProjectDelete {
	return &ProjectDelete{dao: dao}
}

func (service *ProjectDelete) Exec(ctx context.Context, request *ProjectDeleteRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entity, err := service.dao.Exec(ctx, &dao.ProjectDeleteRequest{ID: request.ID, OwnerID: request.OwnerID})
	if errors.Is(err, dao.ErrProjectDeleteNotFound) {
		err = errors.Join(err, ErrProjectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("delete project: %w", err))
	}

	return newProject(entity), nil
}
