package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// ProjectDefaultUpdateDao makes the instance the default of the project. It refuses an instance the project
// does not hold.
type ProjectDefaultUpdateDao interface {
	Exec(ctx context.Context, request *dao.ProjectDefaultUpdateRequest) (*dao.Project, error)
}

// ProjectDefaultUpdateRequest names the instance a project opens on.
type ProjectDefaultUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// InstanceID is the instance that becomes the default of the project. The project must hold it.
	InstanceID uuid.UUID `validate:"required"`
}

// ProjectDefaultUpdate chooses the default instance of a project, and returns the project. The default is a
// field of the project: the choice stamps the project's UpdatedAt, so [ProjectFieldsDrift] reports it, and its
// EditedAt with the lock.
type ProjectDefaultUpdate struct {
	daoLock    ProjectLockDao
	dao        ProjectDefaultUpdateDao
	transactor transaction.Transactor
}

func NewProjectDefaultUpdate(
	daoLock ProjectLockDao, dao ProjectDefaultUpdateDao, transactor transaction.Transactor,
) *ProjectDefaultUpdate {
	return &ProjectDefaultUpdate{daoLock: daoLock, dao: dao, transactor: transactor}
}

func (service *ProjectDefaultUpdate) Exec(ctx context.Context, request *ProjectDefaultUpdateRequest) (*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectDefaultUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("instance.id", request.InstanceID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	now := time.Now()

	var entity *dao.Project

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		entity, err = service.dao.Exec(ctx, &dao.ProjectDefaultUpdateRequest{
			ProjectID:  request.ProjectID,
			InstanceID: request.InstanceID,
			Now:        now,
		})
		if err != nil {
			return fmt.Errorf("update project default: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return newProject(entity), nil
}
