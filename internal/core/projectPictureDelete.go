package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ProjectPictureDeleteDaoLock locks the project its owner names, so the picture removal stamps the project's
// edit.
type ProjectPictureDeleteDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
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

// ProjectPictureDelete removes a picture of a project. A project without the picture keeps none.
type ProjectPictureDelete struct {
	daoLock    ProjectPictureDeleteDaoLock
	dao        ProjectPictureDeleteDao
	transactor transaction.Transactor
}

func NewProjectPictureDelete(
	daoLock ProjectPictureDeleteDaoLock, dao ProjectPictureDeleteDao, transactor transaction.Transactor,
) *ProjectPictureDelete {
	return &ProjectPictureDelete{daoLock: daoLock, dao: dao, transactor: transactor}
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

	var project *Project

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		entity, err := service.dao.Exec(ctx, &dao.ProjectPictureDeleteRequest{
			ProjectID: request.ProjectID,
			Picture:   request.Picture,
		})
		if err != nil {
			return fmt.Errorf("delete picture: %w", err)
		}

		project = newProject(entity)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return project, nil
}
