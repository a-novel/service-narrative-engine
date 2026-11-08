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

// ProjectPictureDeleteDaoOwner checks that the user owns the project.
type ProjectPictureDeleteDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// ProjectPictureDeleteDao removes the picture and its version.
type ProjectPictureDeleteDao interface {
	Exec(ctx context.Context, request *dao.ProjectPictureDeleteRequest) (*dao.Project, error)
}

// ProjectPictureDeleteRequest names the picture to remove.
type ProjectPictureDeleteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Picture names the picture: [ProjectPictureLogo] or [ProjectPictureBackground].
	Picture string `validate:"required,projectpicture"`
}

// ProjectPictureDelete removes a picture of a project. A project without the picture keeps none. A picture is
// cosmetic, so the removal leaves the project's EditedAt as it was.
type ProjectPictureDelete struct {
	daoOwner ProjectPictureDeleteDaoOwner
	dao      ProjectPictureDeleteDao
}

func NewProjectPictureDelete(daoOwner ProjectPictureDeleteDaoOwner, dao ProjectPictureDeleteDao) *ProjectPictureDelete {
	return &ProjectPictureDelete{daoOwner: daoOwner, dao: dao}
}

func (service *ProjectPictureDelete) Exec(ctx context.Context, request *ProjectPictureDeleteRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectPictureDelete")
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

	entity, err := service.dao.Exec(ctx, &dao.ProjectPictureDeleteRequest{
		ProjectID: request.ProjectID,
		Picture:   request.Picture,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("delete picture: %w", err))
	}

	return newProject(entity), nil
}
