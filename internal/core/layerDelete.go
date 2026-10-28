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

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// LayerDeleteDaoOwner checks that the user owns the project.
type LayerDeleteDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// LayerDeleteDaoLock locks the project, so changes to its layers run one at a time.
type LayerDeleteDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// LayerDeleteDaoKind reads the kinds of the layers to delete, and how many layers share each.
type LayerDeleteDaoKind interface {
	Exec(ctx context.Context, request *dao.LayerKindRequest) ([]*dao.LayerKind, error)
}

// LayerDeleteDao removes the layers, and moves the layers after them up.
type LayerDeleteDao interface {
	Exec(ctx context.Context, request *dao.LayerDeleteRequest) ([]*dao.Layer, error)
}

// LayerDeleteRequest names the layers to delete.
type LayerDeleteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// IDs lists the layers to delete, each once, at most 100.
	IDs []uuid.UUID `validate:"required,min=1,max=100,unique,dive,required"`
}

// LayerDelete deletes layers with their cells, and moves the layers after them up to close the gaps. It
// deletes all of them or none, and refuses with [ErrRequiredLayer] to remove the last layer of a
// required kind, such as the manuscript.
type LayerDelete struct {
	daoOwner   LayerDeleteDaoOwner
	daoLock    LayerDeleteDaoLock
	daoKind    LayerDeleteDaoKind
	dao        LayerDeleteDao
	transactor transaction.Transactor
}

func NewLayerDelete(
	daoOwner LayerDeleteDaoOwner,
	daoLock LayerDeleteDaoLock,
	daoKind LayerDeleteDaoKind,
	dao LayerDeleteDao,
	transactor transaction.Transactor,
) *LayerDelete {
	return &LayerDelete{daoOwner: daoOwner, daoLock: daoLock, daoKind: daoKind, dao: dao, transactor: transactor}
}

func (service *LayerDelete) Exec(ctx context.Context, request *LayerDeleteRequest) ([]*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("layers.count", len(request.IDs)),
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

	var layers []*Layer

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		_, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		// The counts are read under the lock, so two deletes cannot both remove the last layer of a kind.
		kinds, err := service.daoKind.Exec(ctx, &dao.LayerKindRequest{ProjectID: request.ProjectID, IDs: request.IDs})
		if err != nil {
			return fmt.Errorf("read layer kinds: %w", err)
		}

		if len(kinds) != len(request.IDs) {
			return fmt.Errorf("%w: %d of %d layers", ErrLayerNotFound, len(request.IDs)-len(kinds), len(request.IDs))
		}

		err = keepRequiredLayers(kinds)
		if err != nil {
			return err
		}

		deleted, err := service.dao.Exec(ctx, &dao.LayerDeleteRequest{ProjectID: request.ProjectID, IDs: request.IDs})
		if err != nil {
			return fmt.Errorf("delete layers: %w", err)
		}

		slices.SortFunc(deleted, func(a, b *dao.Layer) int { return cmp.Compare(a.Position, b.Position) })
		layers = lo.Map(deleted, func(entity *dao.Layer, _ int) *Layer { return newLayer(entity) })

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return layers, nil
}

// keepRequiredLayers returns [ErrRequiredLayer] when the layers to delete hold every layer of a required
// kind their project has.
func keepRequiredLayers(kinds []*dao.LayerKind) error {
	deleting := lo.CountValuesBy(kinds, func(kind *dao.LayerKind) string { return kind.Kind })

	for _, kind := range kinds {
		_, required := requiredLayers[kind.Kind]
		if required && deleting[kind.Kind] == kind.Count {
			return fmt.Errorf("%w: the last %s layer", ErrRequiredLayer, kind.Kind)
		}
	}

	return nil
}
