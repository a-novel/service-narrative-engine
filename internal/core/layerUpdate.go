package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// LayerUpdateDaoLock locks the project its owner names, so the writes to its bytes run one at a time.
type LayerUpdateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// LayerUpdateDao writes the layer's new fields.
type LayerUpdateDao interface {
	Exec(ctx context.Context, request *dao.LayerUpdateRequest) (*dao.Layer, error)
}

// LayerUpdateDaoBytes reads how many bytes the project stores, the write included.
type LayerUpdateDaoBytes interface {
	Exec(ctx context.Context, request *dao.ProjectBytesRequest) (*dao.ProjectBytes, error)
}

// LayerUpdateRequest carries a layer's new fields.
type LayerUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`

	// Name labels the layer's row in the composer.
	Name string `validate:"required,notblank,max=100"`
	// Description says what the layer records.
	Description string `validate:"max=20000"`
}

// LayerUpdate replaces a layer's name and description. It writes nothing when the project would store more
// than [ProjectBytesMax] bytes.
type LayerUpdate struct {
	daoLock    LayerUpdateDaoLock
	dao        LayerUpdateDao
	daoBytes   LayerUpdateDaoBytes
	transactor transaction.Transactor
}

func NewLayerUpdate(
	daoLock LayerUpdateDaoLock,
	dao LayerUpdateDao,
	daoBytes LayerUpdateDaoBytes,
	transactor transaction.Transactor,
) *LayerUpdate {
	return &LayerUpdate{daoLock: daoLock, dao: dao, daoBytes: daoBytes, transactor: transactor}
}

func (service *LayerUpdate) Exec(ctx context.Context, request *LayerUpdateRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerUpdate")
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

		entity, err := service.dao.Exec(ctx, &dao.LayerUpdateRequest{
			ProjectID:   request.ProjectID,
			ID:          request.ID,
			Name:        request.Name,
			Description: request.Description,
			Now:         time.Now(),
		})
		if errors.Is(err, dao.ErrLayerUpdateNotFound) {
			err = errors.Join(err, ErrLayerNotFound)
		}

		if err != nil {
			return fmt.Errorf("update layer: %w", err)
		}

		err = keepProjectBudget(ctx, service.daoBytes, projectBudgetOpts{projectID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("keep project budget: %w", err)
		}

		layer = newLayer(entity)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return layer, nil
}
