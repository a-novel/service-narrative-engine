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

// SceneDeleteDaoOwner checks that the user owns the project.
type SceneDeleteDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// SceneDeleteDaoLock locks the project, so changes to its order run one at a time.
type SceneDeleteDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// SceneDeleteDao removes the scenes, and moves the scenes after them up.
type SceneDeleteDao interface {
	Exec(ctx context.Context, request *dao.SceneDeleteRequest) ([]*dao.Scene, error)
}

// SceneDeleteRequest names the scenes to delete.
type SceneDeleteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// IDs lists the scenes to delete, each once, at most 100.
	IDs []uuid.UUID `validate:"required,min=1,max=100,unique,dive,required"`
}

// SceneDelete deletes scenes with their cells, and moves the scenes after them up to close the gaps. It
// deletes all of them or none.
type SceneDelete struct {
	daoOwner   SceneDeleteDaoOwner
	daoLock    SceneDeleteDaoLock
	dao        SceneDeleteDao
	transactor transaction.Transactor
}

func NewSceneDelete(
	daoOwner SceneDeleteDaoOwner,
	daoLock SceneDeleteDaoLock,
	dao SceneDeleteDao,
	transactor transaction.Transactor,
) *SceneDelete {
	return &SceneDelete{daoOwner: daoOwner, daoLock: daoLock, dao: dao, transactor: transactor}
}

func (service *SceneDelete) Exec(ctx context.Context, request *SceneDeleteRequest) ([]*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("scenes.count", len(request.IDs)),
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

	var scenes []*Scene

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		_, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		deleted, err := service.dao.Exec(ctx, &dao.SceneDeleteRequest{ProjectID: request.ProjectID, IDs: request.IDs})
		if err != nil {
			return fmt.Errorf("delete scenes: %w", err)
		}

		// A miss rolls the whole delete back.
		if len(deleted) != len(request.IDs) {
			return fmt.Errorf("%w: %d of %d scenes", ErrSceneNotFound, len(request.IDs)-len(deleted), len(request.IDs))
		}

		slices.SortFunc(deleted, func(a, b *dao.Scene) int { return cmp.Compare(a.Position, b.Position) })
		scenes = lo.Map(deleted, func(entity *dao.Scene, _ int) *Scene { return newScene(entity) })

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scenes, nil
}
