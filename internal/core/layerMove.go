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

// LayerMoveDaoSelect reads the layer to move and the layer it follows.
type LayerMoveDaoSelect interface {
	Exec(ctx context.Context, request *dao.LayerSelectRequest) (*dao.Layer, error)
}

// LayerMoveDaoLock locks the layer's project, so changes to its order run one at a time.
type LayerMoveDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// LayerMoveDao moves the layer, and shifts the layers between its old and new position.
type LayerMoveDao interface {
	Exec(ctx context.Context, request *dao.LayerMoveRequest) (*dao.Layer, error)
}

// LayerMoveRequest names the layer to move and the layer it follows.
type LayerMoveRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`
	// After is the layer the moved one follows. uuid.Nil places it first.
	After uuid.UUID
}

// LayerMove moves a layer within its project, and returns it at its new position. The layers between its
// old and new position shift by one.
type LayerMove struct {
	daoSelect  LayerMoveDaoSelect
	daoLock    LayerMoveDaoLock
	dao        LayerMoveDao
	transactor transaction.Transactor
}

func NewLayerMove(
	daoSelect LayerMoveDaoSelect,
	daoLock LayerMoveDaoLock,
	dao LayerMoveDao,
	transactor transaction.Transactor,
) *LayerMove {
	return &LayerMove{daoSelect: daoSelect, daoLock: daoLock, dao: dao, transactor: transactor}
}

func (service *LayerMove) Exec(ctx context.Context, request *LayerMoveRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerMove")
	defer span.End()

	span.SetAttributes(
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

	var layer *Layer

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		target, err := service.daoSelect.Exec(ctx, &dao.LayerSelectRequest{ID: request.ID})
		if errors.Is(err, dao.ErrLayerSelectNotFound) {
			err = errors.Join(err, ErrLayerNotFound)
		}

		if err != nil {
			return fmt.Errorf("select layer: %w", err)
		}

		project, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: target.ProjectID})
		if errors.Is(err, dao.ErrProjectLockNotFound) {
			err = errors.Join(err, ErrLayerNotFound)
		}

		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		err = checkOwner(project, request.OwnerID, ErrLayerNotFound)
		if err != nil {
			return err
		}

		// Another change may have moved the layer before the lock, so read its position again.
		target, err = service.daoSelect.Exec(ctx, &dao.LayerSelectRequest{ID: request.ID})
		if errors.Is(err, dao.ErrLayerSelectNotFound) {
			err = errors.Join(err, ErrLayerNotFound)
		}

		if err != nil {
			return fmt.Errorf("select layer: %w", err)
		}

		to := 0

		if request.After != uuid.Nil {
			anchor, err := layerAnchor(ctx, service.daoSelect, project.ID, request.After)
			if err != nil {
				return err
			}

			to = moveTarget(target.Position, anchor.Position)
		}

		if to == target.Position {
			layer = newLayer(target, 0)

			return nil
		}

		moved, err := service.dao.Exec(ctx, &dao.LayerMoveRequest{ID: request.ID, To: to})
		if errors.Is(err, dao.ErrLayerMoveNotFound) {
			err = errors.Join(err, ErrLayerNotFound)
		}

		if err != nil {
			return fmt.Errorf("move layer: %w", err)
		}

		layer = newLayer(moved, 0)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return layer, nil
}
