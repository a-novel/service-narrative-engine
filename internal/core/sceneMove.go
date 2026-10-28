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

// SceneMoveDaoSelect reads the scene to move.
type SceneMoveDaoSelect interface {
	Exec(ctx context.Context, request *dao.SceneSelectRequest) (*dao.Scene, error)
}

// SceneMoveDaoLock locks the scene's project, so changes to its order run one at a time.
type SceneMoveDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// SceneMoveDaoList reads the project's scenes in order.
type SceneMoveDaoList interface {
	Exec(ctx context.Context, request *dao.SceneListRequest) ([]*dao.Scene, error)
}

// SceneMoveDaoReorder writes the new order.
type SceneMoveDaoReorder interface {
	Exec(ctx context.Context, request *dao.SceneReorderRequest) ([]*dao.Scene, error)
}

// SceneMoveRequest names the scene to move and the scene it follows.
type SceneMoveRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	ID      uuid.UUID `validate:"required"`
	// After is the scene the moved one follows. uuid.Nil places it first.
	After uuid.UUID
}

// SceneMove moves a scene within its project. It returns every scene of the project in their new
// order.
type SceneMove struct {
	daoSelect  SceneMoveDaoSelect
	daoLock    SceneMoveDaoLock
	daoList    SceneMoveDaoList
	daoReorder SceneMoveDaoReorder
	transactor transaction.Transactor
}

func NewSceneMove(
	daoSelect SceneMoveDaoSelect,
	daoLock SceneMoveDaoLock,
	daoList SceneMoveDaoList,
	daoReorder SceneMoveDaoReorder,
	transactor transaction.Transactor,
) *SceneMove {
	return &SceneMove{
		daoSelect:  daoSelect,
		daoLock:    daoLock,
		daoList:    daoList,
		daoReorder: daoReorder,
		transactor: transactor,
	}
}

func (service *SceneMove) Exec(ctx context.Context, request *SceneMoveRequest) ([]*Scene, error) {
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

	var scenes []*Scene

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		scene, err := service.daoSelect.Exec(ctx, &dao.SceneSelectRequest{ID: request.ID, OwnerID: request.OwnerID})
		if errors.Is(err, dao.ErrSceneSelectNotFound) {
			err = errors.Join(err, ErrSceneNotFound)
		}

		if err != nil {
			return fmt.Errorf("select scene: %w", err)
		}

		_, err = service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: scene.ProjectID, OwnerID: request.OwnerID})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		current, err := service.daoList.Exec(ctx, &dao.SceneListRequest{ProjectID: scene.ProjectID})
		if err != nil {
			return fmt.Errorf("list scenes: %w", err)
		}

		order, err := moveAfter(lo.Map(current, sceneID), request.ID, request.After)
		if err != nil {
			return err
		}

		reordered, err := service.daoReorder.Exec(ctx, &dao.SceneReorderRequest{ProjectID: scene.ProjectID, IDs: order})
		if err != nil {
			return fmt.Errorf("reorder scenes: %w", err)
		}

		slices.SortFunc(reordered, func(a, b *dao.Scene) int { return cmp.Compare(a.Position, b.Position) })
		scenes = lo.Map(reordered, newScene)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scenes, nil
}
