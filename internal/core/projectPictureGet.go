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

// ProjectPictureGetDaoOwner checks that the user owns the project.
type ProjectPictureGetDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// ProjectPictureGetDao reads the picture.
type ProjectPictureGetDao interface {
	Exec(ctx context.Context, request *dao.ProjectPictureSelectRequest) (*dao.ProjectPicture, error)
}

// ProjectPictureGetRequest names a picture of a project of the caller.
type ProjectPictureGetRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Picture names the picture: [ProjectPictureLogo] or [ProjectPictureBackground].
	Picture string `validate:"required,projectpicture"`
}

// ProjectPictureGet reads a picture of a project of the caller.
type ProjectPictureGet struct {
	daoOwner ProjectPictureGetDaoOwner
	dao      ProjectPictureGetDao
}

func NewProjectPictureGet(daoOwner ProjectPictureGetDaoOwner, dao ProjectPictureGetDao) *ProjectPictureGet {
	return &ProjectPictureGet{daoOwner: daoOwner, dao: dao}
}

func (service *ProjectPictureGet) Exec(
	ctx context.Context, request *ProjectPictureGetRequest,
) (*ProjectPicture, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectPictureGet")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("project.picture", request.Picture),
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

	entity, err := service.dao.Exec(ctx, &dao.ProjectPictureSelectRequest{
		ProjectID: request.ProjectID,
		Picture:   request.Picture,
	})
	if errors.Is(err, dao.ErrProjectPictureSelectNotFound) {
		err = errors.Join(err, ErrPictureNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select picture: %w", err))
	}

	return &ProjectPicture{MediaType: entity.MediaType, Data: entity.Data, Version: entity.UpdatedAt}, nil
}
