package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// LayersUpdateDaoLock locks the project its owner names, so the writes to its bytes run one at a time.
type LayersUpdateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// LayersUpdateDao writes the layers' new fields, all of them or none.
type LayersUpdateDao interface {
	Exec(ctx context.Context, request *dao.LayersUpdateRequest) ([]*dao.Layer, error)
}

// LayersUpdateDaoBytes reads how many bytes the project stores, the write included.
type LayersUpdateDaoBytes interface {
	Exec(ctx context.Context, request *dao.ProjectBytesRequest) (*dao.ProjectBytes, error)
}

// LayersUpdateItem is one layer of a [LayersUpdateRequest].
type LayersUpdateItem struct {
	ID uuid.UUID `validate:"required"`

	// Name labels the layer's row in the composer.
	Name string `validate:"required,notblank,max=100"`
	// Description says what the layer records.
	Description string `validate:"max=20000"`
}

// LayersUpdateRequest carries the new fields of layers.
type LayersUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// Layers lists the layers to update, each once, at most 100.
	Layers []LayersUpdateItem `validate:"required,min=1,max=100,unique=ID,dive"`
}

// LayersUpdate replaces the fields of several layers at once, all of them or none, and returns them in the
// order of the request. [LayerUpdate] updates one. It writes nothing when the project would store more than
// [ProjectBytesMax] bytes.
type LayersUpdate struct {
	daoLock    LayersUpdateDaoLock
	dao        LayersUpdateDao
	daoBytes   LayersUpdateDaoBytes
	transactor transaction.Transactor
}

func NewLayersUpdate(
	daoLock LayersUpdateDaoLock,
	dao LayersUpdateDao,
	daoBytes LayersUpdateDaoBytes,
	transactor transaction.Transactor,
) *LayersUpdate {
	return &LayersUpdate{daoLock: daoLock, dao: dao, daoBytes: daoBytes, transactor: transactor}
}

func (service *LayersUpdate) Exec(ctx context.Context, request *LayersUpdateRequest) ([]*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayersUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("layers.count", len(request.Layers)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	updateItems := make([]dao.LayersUpdateItem, len(request.Layers))

	for i, item := range request.Layers {
		updateItems[i] = dao.LayersUpdateItem{ID: item.ID, Name: item.Name, Description: item.Description}
	}

	var layers []*Layer

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		updated, err := service.dao.Exec(ctx, &dao.LayersUpdateRequest{
			ProjectID: request.ProjectID,
			Layers:    updateItems,
			Now:       time.Now(),
		})
		if err != nil {
			return fmt.Errorf("update layers: %w", err)
		}

		// The update writes none of the layers when the project does not hold one of them.
		if len(updated) != len(request.Layers) {
			return fmt.Errorf(
				"%w: %d of %d layers", ErrLayerNotFound, len(request.Layers)-len(updated), len(request.Layers),
			)
		}

		err = keepProjectBudget(ctx, service.daoBytes, projectBudgetOpts{projectID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("keep project budget: %w", err)
		}

		byID := lo.KeyBy(updated, func(entity *dao.Layer) uuid.UUID { return entity.ID })
		layers = lo.Map(request.Layers, func(item LayersUpdateItem, _ int) *Layer { return newLayer(byID[item.ID]) })

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return layers, nil
}
