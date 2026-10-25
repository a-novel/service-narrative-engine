package core

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// LayerCreateDaoLock locks the project, so changes to its order run one at a time.
type LayerCreateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// LayerCreateDaoList reads the project's layers in order.
type LayerCreateDaoList interface {
	Exec(ctx context.Context, request *dao.LayerListRequest) ([]*dao.Layer, error)
}

// LayerCreateDaoInsert inserts the layer.
type LayerCreateDaoInsert interface {
	Exec(ctx context.Context, request *dao.LayerInsertRequest) (*dao.Layer, error)
}

// LayerCreateDaoReorder moves the layer to its place.
type LayerCreateDaoReorder interface {
	Exec(ctx context.Context, request *dao.LayerReorderRequest) ([]*dao.Layer, error)
}

// LayerCreateRequest carries a new layer and its place in the project.
type LayerCreateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// After is the layer the new one follows. uuid.Nil places it first.
	After       uuid.UUID
	Name        string `validate:"required,notblank,max=100"`
	Description string `validate:"max=1000"`
}

// LayerCreate adds a notes layer to a project, at the place the caller names.
type LayerCreate struct {
	daoLock    LayerCreateDaoLock
	daoList    LayerCreateDaoList
	daoInsert  LayerCreateDaoInsert
	daoReorder LayerCreateDaoReorder
	transactor transaction.Transactor
	config     config.Composer
}

func NewLayerCreate(
	daoLock LayerCreateDaoLock,
	daoList LayerCreateDaoList,
	daoInsert LayerCreateDaoInsert,
	daoReorder LayerCreateDaoReorder,
	transactor transaction.Transactor,
	config config.Composer,
) *LayerCreate {
	return &LayerCreate{
		daoLock:    daoLock,
		daoList:    daoList,
		daoInsert:  daoInsert,
		daoReorder: daoReorder,
		transactor: transactor,
		config:     config,
	}
}

func (service *LayerCreate) Exec(ctx context.Context, request *LayerCreateRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerCreate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("layer.after", request.After.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("generate id: %w", err))
	}

	var layer *Layer

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		_, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: request.ProjectID, OwnerID: request.OwnerID})
		if errors.Is(err, dao.ErrProjectLockNotFound) {
			err = errors.Join(err, ErrProjectNotFound)
		}

		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		layers, err := service.daoList.Exec(ctx, &dao.LayerListRequest{
			ProjectID: request.ProjectID,
			Limit:     service.config.MaxLayers,
		})
		if err != nil {
			return fmt.Errorf("list layers: %w", err)
		}

		if len(layers) >= service.config.MaxLayers {
			return fmt.Errorf("%w: %d layers", ErrProjectFull, len(layers))
		}

		ids := append(lo.Map(layers, layerID), id)

		order, err := moveAfter(ids, id, request.After)
		if err != nil {
			return err
		}

		inserted, err := service.daoInsert.Exec(ctx, &dao.LayerInsertRequest{
			ID:          id,
			ProjectID:   request.ProjectID,
			Position:    len(layers),
			Kind:        LayerKindNotes,
			Name:        request.Name,
			Description: request.Description,
			Now:         time.Now(),
		})
		if err != nil {
			return fmt.Errorf("insert layer: %w", err)
		}

		layer = newLayer(inserted, 0)

		if slices.Equal(order, ids) {
			return nil
		}

		_, err = service.daoReorder.Exec(ctx, &dao.LayerReorderRequest{ProjectID: request.ProjectID, IDs: order})
		if err != nil {
			return fmt.Errorf("reorder layers: %w", err)
		}

		layer.Position = slices.Index(order, id)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return layer, nil
}

// layerID adapts a layer to lo.Map, returning its ID.
func layerID(layer *dao.Layer, _ int) uuid.UUID {
	return layer.ID
}
