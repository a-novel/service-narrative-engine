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

// ScenesCreateDaoLock locks the project its owner names, so changes to its order run one at a time.
type ScenesCreateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// ScenesCreateDaoInsert inserts the scenes, and moves the scenes at or after their position down.
type ScenesCreateDaoInsert interface {
	Exec(ctx context.Context, request *dao.ScenesInsertRequest) ([]*dao.Scene, error)
}

// ScenesCreateItem is one scene of a [ScenesCreateRequest].
type ScenesCreateItem struct {
	// Title names the scene in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Color is the scene's color in the composer, as a #rrggbb hex code in either case.
	Color string `validate:"required,len=7,hexcolor"`
	// Description says what the scene is for, written for the author.
	Description string `validate:"max=20000"`
	// Prompt holds extra instructions for the model when it writes about the scene.
	Prompt string `validate:"max=20000"`
}

// ScenesCreateRequest carries new scenes and their place in the project.
type ScenesCreateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Position is the place of the first new scene in narrative order, from 0, and the others follow it.
	// A position past the last scene appends them.
	Position int `validate:"min=0"`
	// Scenes lists the scenes to add, in narrative order, at most 100.
	Scenes []ScenesCreateItem `validate:"required,min=1,max=100,dive"`
}

// ScenesCreate adds scenes to a project at the position the caller names. The scenes from that
// position on move down to make room.
type ScenesCreate struct {
	daoLock    ScenesCreateDaoLock
	daoInsert  ScenesCreateDaoInsert
	transactor transaction.Transactor
}

func NewScenesCreate(
	daoLock ScenesCreateDaoLock,
	daoInsert ScenesCreateDaoInsert,
	transactor transaction.Transactor,
) *ScenesCreate {
	return &ScenesCreate{daoLock: daoLock, daoInsert: daoInsert, transactor: transactor}
}

func (service *ScenesCreate) Exec(ctx context.Context, request *ScenesCreateRequest) ([]*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ScenesCreate")
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

	ids, err := newIDs(ctx, len(request.Scenes))
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	var scenes []*Scene

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		inserted, err := service.daoInsert.Exec(ctx, &dao.ScenesInsertRequest{
			ProjectID: request.ProjectID,
			Position:  request.Position,
			Scenes: lo.Map(request.Scenes, func(item ScenesCreateItem, index int) dao.ScenesInsertItem {
				return dao.ScenesInsertItem{
					ID: ids[index], Title: item.Title, Color: strings.ToLower(item.Color),
					Description: item.Description, Prompt: item.Prompt,
				}
			}),
			Now: time.Now(),
		})
		if errors.Is(err, dao.ErrScenesInsertFull) {
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
