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

// ProjectDeleteDaoSelect reads the project, to check its owner.
type ProjectDeleteDaoSelect interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

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
	daoSelect ProjectDeleteDaoSelect
	dao       ProjectDeleteDao
}

func NewProjectDelete(daoSelect ProjectDeleteDaoSelect, dao ProjectDeleteDao) *ProjectDelete {
	return &ProjectDelete{daoSelect: daoSelect, dao: dao}
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

	project, err := service.daoSelect.Exec(ctx, &dao.ProjectSelectRequest{ID: request.ID})
	if errors.Is(err, dao.ErrProjectSelectNotFound) {
		err = errors.Join(err, ErrProjectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select project: %w", err))
	}

	err = checkOwner(project, request.OwnerID, ErrProjectNotFound)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	entity, err := service.dao.Exec(ctx, &dao.ProjectDeleteRequest{ID: request.ID})
	if errors.Is(err, dao.ErrProjectDeleteNotFound) {
		err = errors.Join(err, ErrProjectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("delete project: %w", err))
	}

	return newProject(entity), nil
}
