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

// SceneMoveDaoOwner checks that the user owns the project.
type SceneMoveDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// SceneMoveDaoLock locks the project, so changes to its order run one at a time.
type SceneMoveDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// SceneMoveDaoSelect reads the scene to move and the scene it follows.
type SceneMoveDaoSelect interface {
	Exec(ctx context.Context, request *dao.SceneSelectRequest) (*dao.Scene, error)
}

// SceneMoveDao moves the scene, and shifts the scenes between its old and new position.
type SceneMoveDao interface {
	Exec(ctx context.Context, request *dao.SceneMoveRequest) (*dao.Scene, error)
}

// SceneMoveRequest names the scene to move and the scene it follows.
type SceneMoveRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`

	// After is the scene the moved one follows. uuid.Nil places it first.
	After uuid.UUID
}

// SceneMove moves a scene within its project, and returns it at its new position. The scenes between its
// old and new position shift by one.
type SceneMove struct {
	daoOwner   SceneMoveDaoOwner
	daoLock    SceneMoveDaoLock
	daoSelect  SceneMoveDaoSelect
	dao        SceneMoveDao
	transactor transaction.Transactor
}

func NewSceneMove(
	daoOwner SceneMoveDaoOwner,
	daoLock SceneMoveDaoLock,
	daoSelect SceneMoveDaoSelect,
	dao SceneMoveDao,
	transactor transaction.Transactor,
) *SceneMove {
	return &SceneMove{daoOwner: daoOwner, daoLock: daoLock, daoSelect: daoSelect, dao: dao, transactor: transactor}
}

func (service *SceneMove) Exec(ctx context.Context, request *SceneMoveRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneMove")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
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

		// The lock is held, so the position read here holds until the move.
		target, err := service.daoSelect.Exec(ctx, &dao.SceneSelectRequest{ProjectID: request.ProjectID, ID: request.ID})
		if errors.Is(err, dao.ErrSceneSelectNotFound) {
			err = errors.Join(err, ErrSceneNotFound)
		}

		if err != nil {
			return fmt.Errorf("select scene: %w", err)
		}

		to := 0

		if request.After != uuid.Nil {
			anchor, err := service.daoSelect.Exec(ctx, &dao.SceneSelectRequest{
				ProjectID: request.ProjectID, ID: request.After,
			})
			if errors.Is(err, dao.ErrSceneSelectNotFound) {
				err = errors.Join(err, ErrInvalidRequest)
			}

			if err != nil {
				return fmt.Errorf("select anchor: %w", err)
			}

			to = moveTarget(target.Position, anchor.Position)
		}

		if to == target.Position {
			scene = newScene(target)

			return nil
		}

		moved, err := service.dao.Exec(ctx, &dao.SceneMoveRequest{ProjectID: request.ProjectID, ID: request.ID, To: to})
		if errors.Is(err, dao.ErrSceneMoveNotFound) {
			err = errors.Join(err, ErrSceneNotFound)
		}

		if err != nil {
			return fmt.Errorf("move scene: %w", err)
		}

		scene = newScene(moved)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scene, nil
}
