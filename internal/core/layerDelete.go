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

// LayerDeleteDaoLock locks the project its owner names, so changes to its layers run one at a time.
type LayerDeleteDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// LayerDeleteDaoKind reads the kind of the layer to delete, and how many layers share it.
type LayerDeleteDaoKind interface {
	Exec(ctx context.Context, request *dao.LayerKindRequest) ([]*dao.LayerKind, error)
}

// LayerDeleteDao removes the layer, and moves the layers after it up.
type LayerDeleteDao interface {
	Exec(ctx context.Context, request *dao.LayerDeleteRequest) (*dao.Layer, error)
}

// LayerDeleteRequest names the layer to delete.
type LayerDeleteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`
}

// LayerDelete deletes one layer with its cells, and moves the layers after it up by one. It refuses with
// [ErrRequiredLayer] to remove the last layer of a required kind, such as the manuscript. [LayersDelete]
// deletes several at once.
type LayerDelete struct {
	daoLock    LayerDeleteDaoLock
	daoKind    LayerDeleteDaoKind
	dao        LayerDeleteDao
	transactor transaction.Transactor
}

func NewLayerDelete(
	daoLock LayerDeleteDaoLock,
	daoKind LayerDeleteDaoKind,
	dao LayerDeleteDao,
	transactor transaction.Transactor,
) *LayerDelete {
	return &LayerDelete{daoLock: daoLock, daoKind: daoKind, dao: dao, transactor: transactor}
}

func (service *LayerDelete) Exec(ctx context.Context, request *LayerDeleteRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("layer.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	var layer *Layer

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		// The count is read under the lock, so two deletes cannot both remove the last layer of a kind.
		kinds, err := service.daoKind.Exec(ctx, &dao.LayerKindRequest{
			ProjectID: request.ProjectID, IDs: []uuid.UUID{request.ID},
		})
		if err != nil {
			return fmt.Errorf("read layer kind: %w", err)
		}

		if len(kinds) == 0 {
			return ErrLayerNotFound
		}

		err = keepRequiredLayers(kinds)
		if err != nil {
			return err
		}

		deleted, err := service.dao.Exec(ctx, &dao.LayerDeleteRequest{ProjectID: request.ProjectID, ID: request.ID})
		if err != nil {
			return fmt.Errorf("delete layer: %w", err)
		}

		layer = newLayer(deleted)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return layer, nil
}
