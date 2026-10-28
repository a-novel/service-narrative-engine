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

// LayerCreateDaoLock locks the project its owner names, so changes to its order run one at a time.
type LayerCreateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// LayerCreateDaoInsert inserts the layer, and moves the layers at or after its position down.
type LayerCreateDaoInsert interface {
	Exec(ctx context.Context, request *dao.LayerInsertRequest) (*dao.Layer, error)
}

// LayerCreateRequest carries a new layer and its place in the project.
type LayerCreateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Position is the layer's place among the project's rows, from 0. A position past the last layer
	// appends it.
	Position int `validate:"min=0"`
	// Kind names the content schema the layer's cells follow, such as manuscript or notes.
	Kind string `validate:"required,layerkind"`

	// Name labels the layer's row in the composer.
	Name string `validate:"required,notblank,max=100"`
	// Description says what the layer records.
	Description string `validate:"max=20000"`
}

// LayerCreate adds one layer of any kind to a project at the position the caller names. The layers from
// that position on move down by one. [LayersCreate] adds several at once.
type LayerCreate struct {
	daoLock    LayerCreateDaoLock
	daoInsert  LayerCreateDaoInsert
	transactor transaction.Transactor
}

func NewLayerCreate(
	daoLock LayerCreateDaoLock,
	daoInsert LayerCreateDaoInsert,
	transactor transaction.Transactor,
) *LayerCreate {
	return &LayerCreate{daoLock: daoLock, daoInsert: daoInsert, transactor: transactor}
}

func (service *LayerCreate) Exec(ctx context.Context, request *LayerCreateRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerCreate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("layer.position", request.Position),
		attribute.String("layer.kind", request.Kind),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	ids, err := newIDs(ctx, 1)
	if err != nil {
		return nil, otel.ReportError(span, err)
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

		inserted, err := service.daoInsert.Exec(ctx, &dao.LayerInsertRequest{
			ID:          ids[0],
			ProjectID:   request.ProjectID,
			Position:    request.Position,
			Kind:        request.Kind,
			Name:        request.Name,
			Description: request.Description,
			Now:         time.Now(),
		})
		if errors.Is(err, dao.ErrLayerInsertFull) {
			err = errors.Join(err, ErrProjectFull)
		}

		if err != nil {
			return fmt.Errorf("insert layer: %w", err)
		}

		layer = newLayer(inserted)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return layer, nil
}
