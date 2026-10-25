package core

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// SceneCreateDaoLock locks the project, so changes to its order run one at a time.
type SceneCreateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// SceneCreateDaoList reads the project's scenes in order.
type SceneCreateDaoList interface {
	Exec(ctx context.Context, request *dao.SceneListRequest) ([]*dao.Scene, error)
}

// SceneCreateDaoInsert inserts the scene.
type SceneCreateDaoInsert interface {
	Exec(ctx context.Context, request *dao.SceneInsertRequest) ([]*dao.Scene, error)
}

// SceneCreateDaoReorder moves the scene to its place.
type SceneCreateDaoReorder interface {
	Exec(ctx context.Context, request *dao.SceneReorderRequest) ([]*dao.Scene, error)
}

// SceneCreateRequest carries a new scene and its place in the project.
type SceneCreateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// After is the scene the new one follows. uuid.Nil places it first.
	After    uuid.UUID
	Title    string `validate:"max=200"`
	Synopsis string `validate:"max=4000"`
}

// SceneCreate adds a scene to a project, at the place the caller names.
type SceneCreate struct {
	daoLock    SceneCreateDaoLock
	daoList    SceneCreateDaoList
	daoInsert  SceneCreateDaoInsert
	daoReorder SceneCreateDaoReorder
	transactor transaction.Transactor
	config     config.Composer
}

func NewSceneCreate(
	daoLock SceneCreateDaoLock,
	daoList SceneCreateDaoList,
	daoInsert SceneCreateDaoInsert,
	daoReorder SceneCreateDaoReorder,
	transactor transaction.Transactor,
	config config.Composer,
) *SceneCreate {
	return &SceneCreate{
		daoLock:    daoLock,
		daoList:    daoList,
		daoInsert:  daoInsert,
		daoReorder: daoReorder,
		transactor: transactor,
		config:     config,
	}
}

func (service *SceneCreate) Exec(ctx context.Context, request *SceneCreateRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneCreate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("scene.after", request.After.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("generate id: %w", err))
	}

	var scene *Scene

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		_, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: request.ProjectID, OwnerID: request.OwnerID})
		if errors.Is(err, dao.ErrProjectLockNotFound) {
			err = errors.Join(err, ErrProjectNotFound)
		}

		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		scenes, err := service.daoList.Exec(ctx, &dao.SceneListRequest{
			ProjectID: request.ProjectID,
			Limit:     service.config.MaxScenes,
		})
		if err != nil {
			return fmt.Errorf("list scenes: %w", err)
		}

		if len(scenes) >= service.config.MaxScenes {
			return fmt.Errorf("%w: %d scenes", ErrProjectFull, len(scenes))
		}

		ids := append(lo.Map(scenes, sceneID), id)

		order, err := moveAfter(ids, id, request.After)
		if err != nil {
			return err
		}

		inserted, err := service.daoInsert.Exec(ctx, &dao.SceneInsertRequest{
			ProjectID: request.ProjectID,
			Position:  len(scenes),
			Scenes:    []dao.SceneInsertItem{{ID: id, Title: request.Title, Synopsis: request.Synopsis}},
			Now:       time.Now(),
		})
		if err != nil {
			return fmt.Errorf("insert scene: %w", err)
		}

		scene = newScene(inserted[0], 0)

		if slices.Equal(order, ids) {
			return nil
		}

		_, err = service.daoReorder.Exec(ctx, &dao.SceneReorderRequest{ProjectID: request.ProjectID, IDs: order})
		if err != nil {
			return fmt.Errorf("reorder scenes: %w", err)
		}

		scene.Position = slices.Index(order, id)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scene, nil
}

// sceneID adapts a scene to lo.Map, returning its ID.
func sceneID(scene *dao.Scene, _ int) uuid.UUID {
	return scene.ID
}
