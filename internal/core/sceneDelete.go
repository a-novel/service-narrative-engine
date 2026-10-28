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

// SceneDeleteDaoOwner checks that the user owns the project.
type SceneDeleteDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// SceneDeleteDaoLock locks the project, so changes to its order run one at a time.
type SceneDeleteDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// SceneDeleteDao removes the scene, and moves the scenes after it up.
type SceneDeleteDao interface {
	Exec(ctx context.Context, request *dao.SceneDeleteRequest) (*dao.Scene, error)
}

// SceneDeleteRequest names the scene to delete.
type SceneDeleteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`
}

// SceneDelete deletes a scene with its cells, and moves the scenes after it up by one.
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

func (service *SceneDelete) Exec(ctx context.Context, request *SceneDeleteRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("scene.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	owned, err := service.daoOwner.Exec(ctx, &dao.ProjectOwnedRequest{ID: request.ProjectID, OwnerID: request.OwnerID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	if !owned {
		return nil, otel.ReportError(span, ErrProjectNotFound)
	}

	var scene *Scene

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		_, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: request.ProjectID})
		if errors.Is(err, dao.ErrProjectLockNotFound) {
			err = errors.Join(err, ErrProjectNotFound)
		}

		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		deleted, err := service.dao.Exec(ctx, &dao.SceneDeleteRequest{ProjectID: request.ProjectID, ID: request.ID})
		if errors.Is(err, dao.ErrSceneDeleteNotFound) {
			err = errors.Join(err, ErrSceneNotFound)
		}

		if err != nil {
			return fmt.Errorf("delete scene: %w", err)
		}

		scene = newScene(deleted)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scene, nil
}
