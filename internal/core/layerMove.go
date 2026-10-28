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

// LayerMoveDaoLock locks the project its owner names, so changes to its order run one at a time.
type LayerMoveDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// LayerMoveDao moves the layer, and shifts the layers between its old and new position.
type LayerMoveDao interface {
	Exec(ctx context.Context, request *dao.LayerMoveRequest) (*dao.Layer, error)
}

// LayerMoveRequest names the layer to move and its new position.
type LayerMoveRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`

	// Position is the layer's place among the project's rows after the move, from 0. A position past
	// the last layer moves it last.
	Position int `validate:"min=0"`
}

// LayerMove moves a layer within its project, and returns it at its new position. The layers between its
// old and new position shift by one.
type LayerMove struct {
	daoLock    LayerMoveDaoLock
	dao        LayerMoveDao
	transactor transaction.Transactor
}

func NewLayerMove(
	daoLock LayerMoveDaoLock,
	dao LayerMoveDao,
	transactor transaction.Transactor,
) *LayerMove {
	return &LayerMove{daoLock: daoLock, dao: dao, transactor: transactor}
}

func (service *LayerMove) Exec(ctx context.Context, request *LayerMoveRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerMove")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("layer.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("layer.position", request.Position),
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

		moved, err := service.dao.Exec(ctx, &dao.LayerMoveRequest{
			ProjectID: request.ProjectID, ID: request.ID, Position: request.Position,
		})
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
