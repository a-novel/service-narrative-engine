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

// SceneDeleteDaoSelect reads the scene to delete.
type SceneDeleteDaoSelect interface {
	Exec(ctx context.Context, request *dao.SceneSelectRequest) (*dao.Scene, error)
}

// SceneDeleteDaoLock locks the scene's project, so changes to its order run one at a time.
type SceneDeleteDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// SceneDeleteDao removes the scene, and moves the scenes after it up.
type SceneDeleteDao interface {
	Exec(ctx context.Context, request *dao.SceneDeleteRequest) (*dao.Scene, error)
}

// SceneDeleteRequest names the scene to delete.
type SceneDeleteRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`
}

// SceneDelete deletes a scene with its cells, and moves the scenes after it up by one.
type SceneDelete struct {
	daoSelect  SceneDeleteDaoSelect
	daoLock    SceneDeleteDaoLock
	dao        SceneDeleteDao
	transactor transaction.Transactor
}

func NewSceneDelete(
	daoSelect SceneDeleteDaoSelect,
	daoLock SceneDeleteDaoLock,
	dao SceneDeleteDao,
	transactor transaction.Transactor,
) *SceneDelete {
	return &SceneDelete{daoSelect: daoSelect, daoLock: daoLock, dao: dao, transactor: transactor}
}

func (service *SceneDelete) Exec(ctx context.Context, request *SceneDeleteRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("scene.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	var scene *Scene

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		target, err := service.daoSelect.Exec(ctx, &dao.SceneSelectRequest{ID: request.ID})
		if errors.Is(err, dao.ErrSceneSelectNotFound) {
			err = errors.Join(err, ErrSceneNotFound)
		}

		if err != nil {
			return fmt.Errorf("select scene: %w", err)
		}

		project, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: target.ProjectID})
		if errors.Is(err, dao.ErrProjectLockNotFound) {
			err = errors.Join(err, ErrSceneNotFound)
		}

		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		err = checkOwner(project, request.OwnerID, ErrSceneNotFound)
		if err != nil {
			return err
		}

		// The lock serializes deletes, so a miss means a concurrent delete won.
		deleted, err := service.dao.Exec(ctx, &dao.SceneDeleteRequest{ID: request.ID})
		if errors.Is(err, dao.ErrSceneDeleteNotFound) {
			err = errors.Join(err, ErrSceneNotFound)
		}

		if err != nil {
			return fmt.Errorf("delete scene: %w", err)
		}

		scene = newScene(deleted, 0)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scene, nil
}
