package core

import (
	"cmp"
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

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// LayerCreateDaoOwner checks that the user owns the project.
type LayerCreateDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// LayerCreateDaoLock locks the project, so changes to its order run one at a time.
type LayerCreateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// LayerCreateDaoInsert inserts the layers, and moves the layers at or after their position down.
type LayerCreateDaoInsert interface {
	Exec(ctx context.Context, request *dao.LayerInsertRequest) ([]*dao.Layer, error)
}

// LayerCreateItem is one layer of a [LayerCreateRequest].
type LayerCreateItem struct {
	// Kind names the content schema the layer's cells follow, such as manuscript or notes.
	Kind string `validate:"required,layerkind"`

	// Name labels the layer's row in the composer.
	Name string `validate:"required,notblank,max=100"`
	// Description says what the layer records.
	Description string `validate:"max=20000"`
}

// LayerCreateRequest carries new layers and their place in the project.
type LayerCreateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Position is the place of the first new layer among the project's rows, from 0, and the others
	// follow it. A position past the last layer appends them.
	Position int `validate:"min=0"`
	// Layers lists the layers to add, in order, at most 100.
	Layers []LayerCreateItem `validate:"required,min=1,max=100,dive"`
}

// LayerCreate adds layers to a project at the position the caller names. The layers from that
// position on move down to make room.
type LayerCreate struct {
	daoOwner   LayerCreateDaoOwner
	daoLock    LayerCreateDaoLock
	daoInsert  LayerCreateDaoInsert
	transactor transaction.Transactor
}

func NewLayerCreate(
	daoOwner LayerCreateDaoOwner,
	daoLock LayerCreateDaoLock,
	daoInsert LayerCreateDaoInsert,
	transactor transaction.Transactor,
) *LayerCreate {
	return &LayerCreate{daoOwner: daoOwner, daoLock: daoLock, daoInsert: daoInsert, transactor: transactor}
}

func (service *LayerCreate) Exec(ctx context.Context, request *LayerCreateRequest) ([]*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerCreate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("layers.position", request.Position),
		attribute.Int("layers.count", len(request.Layers)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	ids, err := newIDs(ctx, len(request.Layers))
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	var layers []*Layer

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		_, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		inserted, err := service.daoInsert.Exec(ctx, &dao.LayerInsertRequest{
			ProjectID: request.ProjectID,
			Position:  request.Position,
			Layers: lo.Map(request.Layers, func(item LayerCreateItem, index int) dao.LayerInsertItem {
				return dao.LayerInsertItem{ID: ids[index], Kind: item.Kind, Name: item.Name, Description: item.Description}
			}),
			Now: time.Now(),
		})
		if errors.Is(err, dao.ErrLayerInsertFull) {
			err = errors.Join(err, ErrProjectFull)
		}

		if err != nil {
			return fmt.Errorf("insert layers: %w", err)
		}

		slices.SortFunc(inserted, func(a, b *dao.Layer) int { return cmp.Compare(a.Position, b.Position) })
		layers = lo.Map(inserted, func(entity *dao.Layer, _ int) *Layer { return newLayer(entity) })

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return layers, nil
}
