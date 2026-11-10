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

// ProjectLogoDeleteDaoOwner checks that the user owns the project.
type ProjectLogoDeleteDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// ProjectLogoDeleteDao removes the logo and its version.
type ProjectLogoDeleteDao interface {
	Exec(ctx context.Context, request *dao.ProjectLogoDeleteRequest) (*dao.Project, error)
}

// ProjectLogoDeleteRequest names the project whose logo to remove.
type ProjectLogoDeleteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
}

// ProjectLogoDelete removes the logo of a project. A project without a logo stays as it is. A logo is cosmetic,
// so the removal leaves the project's EditedAt as it was.
type ProjectLogoDelete struct {
	daoOwner ProjectLogoDeleteDaoOwner
	dao      ProjectLogoDeleteDao
}

func NewProjectLogoDelete(daoOwner ProjectLogoDeleteDaoOwner, dao ProjectLogoDeleteDao) *ProjectLogoDelete {
	return &ProjectLogoDelete{daoOwner: daoOwner, dao: dao}
}

func (service *ProjectLogoDelete) Exec(ctx context.Context, request *ProjectLogoDeleteRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectLogoDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	entity, err := service.dao.Exec(ctx, &dao.ProjectLogoDeleteRequest{ProjectID: request.ProjectID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("delete logo: %w", err))
	}

	return newProject(entity), nil
}
