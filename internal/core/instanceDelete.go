package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// InstanceDeleteDao removes the instance. It refuses the default instance of the project.
type InstanceDeleteDao interface {
	Exec(ctx context.Context, request *dao.InstanceDeleteRequest) (*dao.InstanceDescriptor, error)
}

// InstanceDeleteRequest names the instance to delete.
type InstanceDeleteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`
}

// InstanceDelete deletes an instance and its content, and returns it without its content. It returns
// [ErrInstanceDefault] for the default instance of the project, which stays until [ProjectDefaultUpdate]
// chooses another. The links that name the instance stay too: the client prunes them.
type InstanceDelete struct {
	daoLock    ProjectLockDao
	dao        InstanceDeleteDao
	transactor transaction.Transactor
}

func NewInstanceDelete(
	daoLock ProjectLockDao, dao InstanceDeleteDao, transactor transaction.Transactor,
) *InstanceDelete {
	return &InstanceDelete{daoLock: daoLock, dao: dao, transactor: transactor}
}

func (service *InstanceDelete) Exec(ctx context.Context, request *InstanceDeleteRequest) (*InstanceDescriptor, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.InstanceDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("instance.id", request.ID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	var entity *dao.InstanceDescriptor

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		entity, err = service.dao.Exec(ctx, &dao.InstanceDeleteRequest{ProjectID: request.ProjectID, ID: request.ID})
		if err != nil {
			return fmt.Errorf("delete instance: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return newInstanceDescriptor(entity), nil
}
