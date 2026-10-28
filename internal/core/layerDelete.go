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

// LayerDeleteDaoSelect reads the layer to delete.
type LayerDeleteDaoSelect interface {
	Exec(ctx context.Context, request *dao.LayerSelectRequest) (*dao.Layer, error)
}

// LayerDeleteDaoLock locks the layer's project, so changes to its order run one at a time.
type LayerDeleteDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// LayerDeleteDao removes the layer, and moves the layers after it up.
type LayerDeleteDao interface {
	Exec(ctx context.Context, request *dao.LayerDeleteRequest) (*dao.Layer, error)
}

// LayerDeleteRequest names the layer to delete.
type LayerDeleteRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`
}

// LayerDelete deletes a notes layer with its cells, and moves the layers after it up by one. It
// refuses the manuscript layer with [ErrManuscriptLayer].
type LayerDelete struct {
	daoSelect  LayerDeleteDaoSelect
	daoLock    LayerDeleteDaoLock
	dao        LayerDeleteDao
	transactor transaction.Transactor
}

func NewLayerDelete(
	daoSelect LayerDeleteDaoSelect,
	daoLock LayerDeleteDaoLock,
	dao LayerDeleteDao,
	transactor transaction.Transactor,
) *LayerDelete {
	return &LayerDelete{daoSelect: daoSelect, daoLock: daoLock, dao: dao, transactor: transactor}
}

func (service *LayerDelete) Exec(ctx context.Context, request *LayerDeleteRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("layer.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
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

		if target.Kind == LayerKindManuscript {
			return ErrManuscriptLayer
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

		// The lock serializes deletes, so a miss means a concurrent delete won.
		deleted, err := service.dao.Exec(ctx, &dao.LayerDeleteRequest{ID: request.ID})
		if errors.Is(err, dao.ErrLayerDeleteNotFound) {
			err = errors.Join(err, ErrLayerNotFound)
		}

		if err != nil {
			return fmt.Errorf("delete layer: %w", err)
		}

		layer = newLayer(deleted, 0)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return layer, nil
}
