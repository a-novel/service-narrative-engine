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

// InstanceUpdateDao renames the instance.
type InstanceUpdateDao interface {
	Exec(ctx context.Context, request *dao.InstanceUpdateRequest) (*dao.InstanceDescriptor, error)
}

// InstanceUpdateRequest carries the new name of an instance.
type InstanceUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`

	// Name is the display name the writer gives the instance, [InstanceNameMax] characters at most. It may be
	// empty, and the client then shows a name of its own. A name of only whitespace is refused: it would show
	// as none, and no client would replace it.
	Name string `validate:"omitempty,notblank,max=80"`
}

// InstanceUpdate renames an instance, or clears its name. The name is the one field of an instance a writer
// edits outside its plugin, and a rename leaves the instance's UpdatedAt, so it refuses no content write. It
// writes nothing when the project would store more than [ProjectBytesMax] bytes.
type InstanceUpdate struct {
	daoLock    ProjectLockDao
	dao        InstanceUpdateDao
	daoBytes   ProjectBytesDao
	transactor transaction.Transactor
}

func NewInstanceUpdate(
	daoLock ProjectLockDao,
	dao InstanceUpdateDao,
	daoBytes ProjectBytesDao,
	transactor transaction.Transactor,
) *InstanceUpdate {
	return &InstanceUpdate{daoLock: daoLock, dao: dao, daoBytes: daoBytes, transactor: transactor}
}

func (service *InstanceUpdate) Exec(ctx context.Context, request *InstanceUpdateRequest) (*InstanceDescriptor, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.InstanceUpdate")
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

		entity, err = service.dao.Exec(ctx, &dao.InstanceUpdateRequest{
			ProjectID: request.ProjectID,
			ID:        request.ID,
			Name:      request.Name,
		})
		if err != nil {
			return fmt.Errorf("update instance: %w", err)
		}

		err = keepProjectBudget(ctx, service.daoBytes, projectBudgetOpts{projectID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("keep project budget: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return newInstanceDescriptor(entity), nil
}
