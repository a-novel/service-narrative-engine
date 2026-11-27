package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
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

// ProjectDelete deletes a project with its logo.
type ProjectDelete struct {
	daoOwner ProjectOwnerDao
	dao      ProjectDeleteDao
}

func NewProjectDelete(daoOwner ProjectOwnerDao, dao ProjectDeleteDao) *ProjectDelete {
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
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("delete project: %w", err))
	}

	return newProject(entity), nil
}
