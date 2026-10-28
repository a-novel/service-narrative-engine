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

// LayerDeleteDaoOwner checks that the user owns the project.
type LayerDeleteDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// LayerDeleteDaoKind reads the kind of the layer to delete, which may name a required layer.
type LayerDeleteDaoKind interface {
	Exec(ctx context.Context, request *dao.LayerKindRequest) (string, error)
}

// LayerDeleteDaoLock locks the project, so changes to its order run one at a time.
type LayerDeleteDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// LayerDeleteDao removes the layer, and moves the layers after it up.
type LayerDeleteDao interface {
	Exec(ctx context.Context, request *dao.LayerDeleteRequest) (*dao.Layer, error)
}

// LayerDeleteRequest names the layer to delete.
type LayerDeleteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`
}

// LayerDelete deletes a notes layer with its cells, and moves the layers after it up by one. It
// refuses a required layer, such as the manuscript, with [ErrRequiredLayer].
type LayerDelete struct {
	daoOwner   LayerDeleteDaoOwner
	daoKind    LayerDeleteDaoKind
	daoLock    LayerDeleteDaoLock
	dao        LayerDeleteDao
	transactor transaction.Transactor
}

func NewLayerDelete(
	daoOwner LayerDeleteDaoOwner,
	daoKind LayerDeleteDaoKind,
	daoLock LayerDeleteDaoLock,
	dao LayerDeleteDao,
	transactor transaction.Transactor,
) *LayerDelete {
	return &LayerDelete{
		daoOwner: daoOwner, daoKind: daoKind, daoLock: daoLock, dao: dao, transactor: transactor,
	}
}

func (service *LayerDelete) Exec(ctx context.Context, request *LayerDeleteRequest) (*Layer, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerDelete")
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

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	kind, err := service.daoKind.Exec(ctx, &dao.LayerKindRequest{ProjectID: request.ProjectID, ID: request.ID})
	if errors.Is(err, dao.ErrLayerKindNotFound) {
		err = errors.Join(err, ErrLayerNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("read layer kind: %w", err))
	}

	if _, required := requiredLayers[kind]; required {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %s", ErrRequiredLayer, kind))
	}

	var layer *Layer

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		_, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		deleted, err := service.dao.Exec(ctx, &dao.LayerDeleteRequest{ProjectID: request.ProjectID, ID: request.ID})
		if err != nil {
			return fmt.Errorf("delete layer: %w", err)
		}

		layer = newLayer(deleted)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return layer, nil
}
