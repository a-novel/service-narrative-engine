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

// SceneDeleteDaoLock locks the project its owner names, so changes to its order run one at a time.
type SceneDeleteDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// SceneDeleteDao removes the scene, and moves the scenes after it in its beat up.
type SceneDeleteDao interface {
	Exec(ctx context.Context, request *dao.SceneDeleteRequest) (*dao.Scene, error)
}

// SceneDeleteRequest names the scene to delete.
type SceneDeleteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`
}

// SceneDelete deletes one scene with its cells, and moves the scenes after it in its beat up by one.
// [ScenesDelete] deletes several at once.
type SceneDelete struct {
	daoLock    SceneDeleteDaoLock
	dao        SceneDeleteDao
	transactor transaction.Transactor
}

func NewSceneDelete(daoLock SceneDeleteDaoLock, dao SceneDeleteDao, transactor transaction.Transactor) *SceneDelete {
	return &SceneDelete{daoLock: daoLock, dao: dao, transactor: transactor}
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

	var scene *Scene

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
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
