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

// LayerCreateDaoLock locks the project, so changes to its order run one at a time.
type LayerCreateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// LayerCreateDaoSelect reads the layer the new one follows.
type LayerCreateDaoSelect interface {
	Exec(ctx context.Context, request *dao.LayerSelectRequest) (*dao.Layer, error)
}

// LayerCreateDaoInsert inserts the layer, and moves the layers at or after its position down.
type LayerCreateDaoInsert interface {
	Exec(ctx context.Context, request *dao.LayerInsertRequest) (*dao.Layer, error)
}

// LayerCreateRequest carries a new layer and its place in the project.
type LayerCreateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// After is the layer the new one follows. uuid.Nil places it first.
	After       uuid.UUID
	Name        string `validate:"required,notblank,max=100"`
	Description string `validate:"max=20000"`
}

// LayerCreate adds a notes layer to a project, right after the layer the caller names.
type LayerCreate struct {
	daoLock    LayerCreateDaoLock
	daoSelect  LayerCreateDaoSelect
	daoInsert  LayerCreateDaoInsert
	transactor transaction.Transactor
}

func NewLayerCreate(
	daoLock LayerCreateDaoLock,
	daoSelect LayerCreateDaoSelect,
	daoInsert LayerCreateDaoInsert,
	transactor transaction.Transactor,
) *LayerCreate {
	return &LayerCreate{daoLock: daoLock, daoSelect: daoSelect, daoInsert: daoInsert, transactor: transactor}
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
		project, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: request.ProjectID})
		if errors.Is(err, dao.ErrProjectLockNotFound) {
			err = errors.Join(err, ErrProjectNotFound)
		}

		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		err = checkOwner(project, request.OwnerID, ErrProjectNotFound)
		if err != nil {
			return err
		}

		position := 0

		if request.After != uuid.Nil {
			anchor, err := layerAnchor(ctx, service.daoSelect, project.ID, request.After)
			if err != nil {
				return err
			}

			position = anchor.Position + 1
		}

		inserted, err := service.daoInsert.Exec(ctx, &dao.LayerInsertRequest{
			ID:          id,
			ProjectID:   request.ProjectID,
			Position:    position,
			Kind:        LayerKindNotes,
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

		layer = newLayer(inserted, 0)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return layer, nil
}
