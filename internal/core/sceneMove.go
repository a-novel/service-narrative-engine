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

// SceneMoveDao moves the scene, and shifts the scenes between its old and new position.
type SceneMoveDao interface {
	Exec(ctx context.Context, request *dao.SceneMoveRequest) (*dao.Scene, error)
}

// SceneMoveRequest names the scene to move and its new position.
type SceneMoveRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`

	// Position is the scene's place in narrative order after the move, from 0. A position past the last
	// scene moves it last.
	Position int `validate:"min=0"`
}

// SceneMove moves a scene within its project, and returns it at its new position. The scenes between its
// old and new position shift by one.
type SceneMove struct {
	daoOwner   SceneMoveDaoOwner
	daoLock    SceneMoveDaoLock
	dao        SceneMoveDao
	transactor transaction.Transactor
}

func NewSceneMove(
	daoOwner SceneMoveDaoOwner,
	daoLock SceneMoveDaoLock,
	dao SceneMoveDao,
	transactor transaction.Transactor,
) *SceneMove {
	return &SceneMove{daoOwner: daoOwner, daoLock: daoLock, dao: dao, transactor: transactor}
}

func (service *SceneMove) Exec(ctx context.Context, request *SceneMoveRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneMove")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("scene.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("scene.position", request.Position),
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

	var scene *Scene

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		_, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		moved, err := service.dao.Exec(ctx, &dao.SceneMoveRequest{
			ProjectID: request.ProjectID, ID: request.ID, Position: request.Position,
		})
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
