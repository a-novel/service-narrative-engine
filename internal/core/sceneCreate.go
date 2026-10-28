package core

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
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

// SceneCreateDaoInsert inserts the scenes, and moves the scenes at or after their position down.
type SceneCreateDaoInsert interface {
	Exec(ctx context.Context, request *dao.SceneInsertRequest) ([]*dao.Scene, error)
}

// SceneCreateItem is one scene of a [SceneCreateRequest].
type SceneCreateItem struct {
	// Title names the scene in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Color is the scene's color in the composer, as a #rrggbb hex code in either case.
	Color string `validate:"required,len=7,hexcolor"`
	// Description says what the scene is for, written for the author.
	Description string `validate:"max=20000"`
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt string `validate:"max=20000"`
}

// SceneCreateRequest carries new scenes and their place in the project.
type SceneCreateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Position is the place of the first new scene in narrative order, from 0, and the others follow it.
	// A position past the last scene appends them.
	Position int `validate:"min=0"`
	// Scenes lists the scenes to add, in narrative order, at most 100.
	Scenes []SceneCreateItem `validate:"required,min=1,max=100,dive"`
}

// SceneCreate adds scenes to a project at the position the caller names. The scenes from that
// position on move down to make room.
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

func (service *SceneCreate) Exec(ctx context.Context, request *SceneCreateRequest) ([]*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneCreate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("scenes.position", request.Position),
		attribute.Int("scenes.count", len(request.Scenes)),
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

	ids, err := newIDs(ctx, len(request.Scenes))
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	var scenes []*Scene

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		_, err := service.daoLock.Exec(ctx, &dao.ProjectLockRequest{ID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		inserted, err := service.daoInsert.Exec(ctx, &dao.SceneInsertRequest{
			ProjectID: request.ProjectID,
			Position:  request.Position,
			Scenes: lo.Map(request.Scenes, func(item SceneCreateItem, index int) dao.SceneInsertItem {
				return dao.SceneInsertItem{
					ID: ids[index], Title: item.Title, Color: strings.ToLower(item.Color),
					Description: item.Description, Prompt: item.Prompt,
				}
			}),
			Now: time.Now(),
		})
		if errors.Is(err, dao.ErrSceneInsertFull) {
			err = errors.Join(err, ErrProjectFull)
		}

		if err != nil {
			return fmt.Errorf("insert scenes: %w", err)
		}

		slices.SortFunc(inserted, func(a, b *dao.Scene) int { return cmp.Compare(a.Position, b.Position) })
		scenes = lo.Map(inserted, func(entity *dao.Scene, _ int) *Scene { return newScene(entity) })

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scenes, nil
}
