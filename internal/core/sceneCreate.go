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

// SceneCreateDaoOwner checks that the user owns the project.
type SceneCreateDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// SceneCreateDaoLock locks the project, so changes to its order run one at a time.
type SceneCreateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (*dao.Project, error)
}

// SceneCreateDaoInsert inserts the scene, and moves the scenes at or after its position down.
type SceneCreateDaoInsert interface {
	Exec(ctx context.Context, request *dao.SceneInsertRequest) ([]*dao.Scene, error)
}

// SceneCreateRequest carries a new scene and its place in the project.
type SceneCreateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Position is the scene's place in narrative order, from 0. A position past the last scene appends
	// it.
	Position int `validate:"min=0"`

	// Title names the scene in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Color is the scene's color in the composer, as a #rrggbb hex code in either case.
	Color string `validate:"required,len=7,hexcolor"`
	// Description says what the scene is for, written for the author.
	Description string `validate:"max=20000"`
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt string `validate:"max=20000"`
}

// SceneCreate adds a scene to a project at the position the caller names. The scenes from that
// position on move down by one.
type SceneCreate struct {
	daoOwner   SceneCreateDaoOwner
	daoLock    SceneCreateDaoLock
	daoInsert  SceneCreateDaoInsert
	transactor transaction.Transactor
}

func NewSceneCreate(
	daoOwner SceneCreateDaoOwner,
	daoLock SceneCreateDaoLock,
	daoInsert SceneCreateDaoInsert,
	transactor transaction.Transactor,
) *SceneCreate {
	return &SceneCreate{daoOwner: daoOwner, daoLock: daoLock, daoInsert: daoInsert, transactor: transactor}
}

func (service *SceneCreate) Exec(ctx context.Context, request *SceneCreateRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneCreate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
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

	id, err := uuid.NewV7()
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("generate id: %w", err))
	}

	var scene *Scene

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		_, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		inserted, err := service.daoInsert.Exec(ctx, &dao.SceneInsertRequest{
			ProjectID: request.ProjectID,
			Position:  request.Position,
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

		scene = newScene(inserted[0])

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scene, nil
}
