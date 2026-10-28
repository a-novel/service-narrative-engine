package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// SceneCreateDaoLock locks the project, so changes to its order run one at a time.
type SceneCreateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// SceneCreateDaoSelect reads the scene the new one follows.
type SceneCreateDaoSelect interface {
	Exec(ctx context.Context, request *dao.SceneSelectRequest) (*dao.Scene, error)
}

// SceneCreateDaoInsert inserts the scene, and moves the scenes at or after its position down.
type SceneCreateDaoInsert interface {
	Exec(ctx context.Context, request *dao.SceneInsertRequest) ([]*dao.Scene, error)
}

// SceneCreateRequest carries a new scene and its place in the project.
type SceneCreateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// After is the scene the new one follows. uuid.Nil places it first.
	After uuid.UUID
	Title string `validate:"required,notblank,max=200"`
	// Color is the scene's color in the composer, as a #rrggbb hex code in either case.
	Color       string `validate:"required,len=7,hexcolor"`
	Description string `validate:"max=20000"`
	Prompt      string `validate:"max=20000"`
}

// SceneCreate adds a scene to a project, right after the scene the caller names.
type SceneCreate struct {
	daoLock    SceneCreateDaoLock
	daoSelect  SceneCreateDaoSelect
	daoInsert  SceneCreateDaoInsert
	transactor transaction.Transactor
}

func NewSceneCreate(
	daoLock SceneCreateDaoLock,
	daoSelect SceneCreateDaoSelect,
	daoInsert SceneCreateDaoInsert,
	transactor transaction.Transactor,
) *SceneCreate {
	return &SceneCreate{daoLock: daoLock, daoSelect: daoSelect, daoInsert: daoInsert, transactor: transactor}
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
		project, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: request.ProjectID})
		if errors.Is(err, dao.ErrProjectLockNotFound) {
			err = errors.Join(err, ErrProjectNotFound)
		}

		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		err = checkOwner(project, request.OwnerID, ErrProjectNotFound)
		if err != nil {
			return err
		}

		position := 0

		if request.After != uuid.Nil {
			anchor, err := sceneAnchor(ctx, service.daoSelect, project.ID, request.After)
			if err != nil {
				return err
			}

			position = anchor.Position + 1
		}

		inserted, err := service.daoInsert.Exec(ctx, &dao.SceneInsertRequest{
			ProjectID: request.ProjectID,
			Position:  position,
			Scenes: []dao.SceneInsertItem{{
				ID: id, Title: request.Title, Color: strings.ToLower(request.Color),
				Description: request.Description, Prompt: request.Prompt,
			}},
			Now: time.Now(),
		})
		if errors.Is(err, dao.ErrSceneInsertFull) {
			err = errors.Join(err, ErrProjectFull)
		}

		if err != nil {
			return fmt.Errorf("insert scene: %w", err)
		}

		scene = newScene(inserted[0], 0)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scene, nil
}
