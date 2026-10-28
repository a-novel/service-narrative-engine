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

// ProjectDeleteDaoOwner checks that the user owns the project.
type ProjectDeleteDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
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
	daoOwner ProjectDeleteDaoOwner
	dao      ProjectDeleteDao
}

func NewProjectDelete(daoOwner ProjectDeleteDaoOwner, dao ProjectDeleteDao) *ProjectDelete {
	return &ProjectDelete{daoOwner: daoOwner, dao: dao}
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

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
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
