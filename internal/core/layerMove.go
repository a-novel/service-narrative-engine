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

// LayerMoveDaoOwner checks that the user owns the project.
type LayerMoveDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// LayerMoveDaoLock locks the project, so changes to its order run one at a time.
type LayerMoveDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// LayerMoveDaoSelect reads the layer to move and the layer it follows.
type LayerMoveDaoSelect interface {
	Exec(ctx context.Context, request *dao.LayerSelectRequest) (*dao.Layer, error)
}

// LayerMoveDao moves the layer, and shifts the layers between its old and new position.
type LayerMoveDao interface {
	Exec(ctx context.Context, request *dao.LayerMoveRequest) (*dao.Layer, error)
}

// LayerMoveRequest names the layer to move and the layer it follows.
type LayerMoveRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`

	// After is the layer the moved one follows. uuid.Nil places it first.
	After uuid.UUID
}

// LayerMove moves a layer within its project, and returns it at its new position. The layers between its
// old and new position shift by one.
type LayerMove struct {
	daoOwner   LayerMoveDaoOwner
	daoLock    LayerMoveDaoLock
	daoSelect  LayerMoveDaoSelect
	dao        LayerMoveDao
	transactor transaction.Transactor
}

func NewLayerMove(
	daoOwner LayerMoveDaoOwner,
	daoLock LayerMoveDaoLock,
	daoSelect LayerMoveDaoSelect,
	dao LayerMoveDao,
	transactor transaction.Transactor,
) *LayerMove {
	return &LayerMove{daoOwner: daoOwner, daoLock: daoLock, daoSelect: daoSelect, dao: dao, transactor: transactor}
}

func (service *LayerMove) Exec(ctx context.Context, request *LayerMoveRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerMove")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("layer.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("layer.after", request.After.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	if request.After == request.ID {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %s cannot follow itself", ErrInvalidRequest, request.ID))
	}

	owned, err := service.daoOwner.Exec(ctx, &dao.ProjectOwnedRequest{ID: request.ProjectID, OwnerID: request.OwnerID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	if !owned {
		return nil, otel.ReportError(span, ErrProjectNotFound)
	}

	var layer *Layer

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		_, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: request.ProjectID})
		if errors.Is(err, dao.ErrProjectLockNotFound) {
			err = errors.Join(err, ErrProjectNotFound)
		}

		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		// The lock is held, so the position read here holds until the move.
		target, err := service.daoSelect.Exec(ctx, &dao.LayerSelectRequest{ProjectID: request.ProjectID, ID: request.ID})
		if errors.Is(err, dao.ErrLayerSelectNotFound) {
			err = errors.Join(err, ErrLayerNotFound)
		}

		if err != nil {
			return fmt.Errorf("select layer: %w", err)
		}

		to := 0

		if request.After != uuid.Nil {
			anchor, err := service.daoSelect.Exec(ctx, &dao.LayerSelectRequest{
				ProjectID: request.ProjectID, ID: request.After,
			})
			if errors.Is(err, dao.ErrLayerSelectNotFound) {
				err = errors.Join(err, ErrInvalidRequest)
			}

			if err != nil {
				return fmt.Errorf("select anchor: %w", err)
			}

			to = moveTarget(target.Position, anchor.Position)
		}

		if to == target.Position {
			layer = newLayer(target)

			return nil
		}

		moved, err := service.dao.Exec(ctx, &dao.LayerMoveRequest{ProjectID: request.ProjectID, ID: request.ID, To: to})
		if errors.Is(err, dao.ErrLayerMoveNotFound) {
			err = errors.Join(err, ErrLayerNotFound)
		}

		if err != nil {
			return fmt.Errorf("move layer: %w", err)
		}

		layer = newLayer(moved)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return layer, nil
}
