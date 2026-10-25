package core

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// LayerMoveDaoSelect reads the layer to move.
type LayerMoveDaoSelect interface {
	Exec(ctx context.Context, request *dao.LayerSelectRequest) (*dao.Layer, error)
}

// LayerMoveDaoLock locks the layer's project, so changes to its order run one at a time.
type LayerMoveDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// LayerMoveDaoList reads the project's layers in order.
type LayerMoveDaoList interface {
	Exec(ctx context.Context, request *dao.LayerListRequest) ([]*dao.Layer, error)
}

// LayerMoveDaoReorder writes the new order.
type LayerMoveDaoReorder interface {
	Exec(ctx context.Context, request *dao.LayerReorderRequest) ([]*dao.Layer, error)
}

// LayerMoveRequest names the layer to move and the layer it follows.
type LayerMoveRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`
	// After is the layer the moved one follows. uuid.Nil places it first.
	After uuid.UUID
}

// LayerMove moves a layer within its project. It returns every layer of the project in their new
// order. A move leaves the revisions unchanged.
type LayerMove struct {
	daoSelect  LayerMoveDaoSelect
	daoLock    LayerMoveDaoLock
	daoList    LayerMoveDaoList
	daoReorder LayerMoveDaoReorder
	transactor transaction.Transactor
	config     config.Composer
}

func NewLayerMove(
	daoSelect LayerMoveDaoSelect,
	daoLock LayerMoveDaoLock,
	daoList LayerMoveDaoList,
	daoReorder LayerMoveDaoReorder,
	transactor transaction.Transactor,
	config config.Composer,
) *LayerMove {
	return &LayerMove{
		daoSelect:  daoSelect,
		daoLock:    daoLock,
		daoList:    daoList,
		daoReorder: daoReorder,
		transactor: transactor,
		config:     config,
	}
}

func (service *LayerMove) Exec(ctx context.Context, request *LayerMoveRequest) ([]*Layer, error) {
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

	var layers []*Layer

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		layer, err := service.daoSelect.Exec(ctx, &dao.LayerSelectRequest{ID: request.ID, OwnerID: request.OwnerID})
		if errors.Is(err, dao.ErrLayerSelectNotFound) {
			err = errors.Join(err, ErrLayerNotFound)
		}

		if err != nil {
			return fmt.Errorf("select layer: %w", err)
		}

		_, err = service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: layer.ProjectID, OwnerID: request.OwnerID})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		current, err := service.daoList.Exec(ctx, &dao.LayerListRequest{
			ProjectID: layer.ProjectID,
			Limit:     service.config.MaxLayers,
		})
		if err != nil {
			return fmt.Errorf("list layers: %w", err)
		}

		order, err := moveAfter(lo.Map(current, layerID), request.ID, request.After)
		if err != nil {
			return err
		}

		reordered, err := service.daoReorder.Exec(ctx, &dao.LayerReorderRequest{ProjectID: layer.ProjectID, IDs: order})
		if err != nil {
			return fmt.Errorf("reorder layers: %w", err)
		}

		slices.SortFunc(reordered, func(a, b *dao.Layer) int { return cmp.Compare(a.Position, b.Position) })
		layers = lo.Map(reordered, newLayer)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return layers, nil
}
