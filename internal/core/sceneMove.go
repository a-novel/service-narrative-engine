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

// SceneMoveDaoSelect reads the scene to move and the scene it follows.
type SceneMoveDaoSelect interface {
	Exec(ctx context.Context, request *dao.SceneSelectRequest) (*dao.Scene, error)
}

// SceneMoveDaoLock locks the scene's project, so changes to its order run one at a time.
type SceneMoveDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// SceneMoveDao moves the scene, and shifts the scenes between its old and new position.
type SceneMoveDao interface {
	Exec(ctx context.Context, request *dao.SceneMoveRequest) (*dao.Scene, error)
}

// SceneMoveRequest names the scene to move and the scene it follows.
type SceneMoveRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`
	// After is the scene the moved one follows. uuid.Nil places it first.
	After uuid.UUID
}

// SceneMove moves a scene within its project, and returns it at its new position. The scenes between its
// old and new position shift by one.
type SceneMove struct {
	daoSelect  SceneMoveDaoSelect
	daoLock    SceneMoveDaoLock
	dao        SceneMoveDao
	transactor transaction.Transactor
}

func NewSceneMove(
	daoSelect SceneMoveDaoSelect,
	daoLock SceneMoveDaoLock,
	dao SceneMoveDao,
	transactor transaction.Transactor,
) *SceneMove {
	return &SceneMove{daoSelect: daoSelect, daoLock: daoLock, dao: dao, transactor: transactor}
}

func (service *SceneMove) Exec(ctx context.Context, request *SceneMoveRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneMove")
	defer span.End()

	span.SetAttributes(
		attribute.String("scene.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("scene.after", request.After.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	if request.After == request.ID {
		return nil, otel.ReportError(span, fmt.Errorf("%w: %s cannot follow itself", ErrInvalidRequest, request.ID))
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

		// Another change may have moved the scene before the lock, so read its position again.
		target, err = service.daoSelect.Exec(ctx, &dao.SceneSelectRequest{ID: request.ID})
		if errors.Is(err, dao.ErrSceneSelectNotFound) {
			err = errors.Join(err, ErrSceneNotFound)
		}

		if err != nil {
			return fmt.Errorf("select scene: %w", err)
		}

		to := 0

		if request.After != uuid.Nil {
			anchor, err := sceneAnchor(ctx, service.daoSelect, project.ID, request.After)
			if err != nil {
				return err
			}

			to = moveTarget(target.Position, anchor.Position)
		}

		if to == target.Position {
			scene = newScene(target, 0)

			return nil
		}

		moved, err := service.dao.Exec(ctx, &dao.SceneMoveRequest{ID: request.ID, To: to})
		if errors.Is(err, dao.ErrSceneMoveNotFound) {
			err = errors.Join(err, ErrSceneNotFound)
		}

		if err != nil {
			return fmt.Errorf("move scene: %w", err)
		}

		scene = newScene(moved, 0)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scene, nil
}
