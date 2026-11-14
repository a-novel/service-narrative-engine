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

// ProjectLogoGetDaoOwner checks that the user owns the project.
type ProjectLogoGetDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// ProjectLogoGetDao reads the logo.
type ProjectLogoGetDao interface {
	Exec(ctx context.Context, request *dao.ProjectLogoSelectRequest) (*dao.ProjectLogo, error)
}

// ProjectLogoGetRequest names a project of the caller.
type ProjectLogoGetRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
}

// ProjectLogoGet reads the logo of a project of the caller.
type ProjectLogoGet struct {
	daoOwner ProjectLogoGetDaoOwner
	dao      ProjectLogoGetDao
}

func NewProjectLogoGet(daoOwner ProjectLogoGetDaoOwner, dao ProjectLogoGetDao) *ProjectLogoGet {
	return &ProjectLogoGet{daoOwner: daoOwner, dao: dao}
}

func (service *ProjectLogoGet) Exec(
	ctx context.Context, request *ProjectLogoGetRequest,
) (*ProjectLogo, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectLogoGet")
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

	entity, err := service.dao.Exec(ctx, &dao.ProjectLogoSelectRequest{ProjectID: request.ProjectID})
	if errors.Is(err, dao.ErrProjectLogoSelectNotFound) {
		err = errors.Join(err, ErrProjectLogoNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select logo: %w", err))
	}

	return &ProjectLogo{
		MediaType: entity.MediaType, Data: entity.Data, Checksum: entity.Checksum, UpdatedAt: entity.UpdatedAt,
	}, nil
}
