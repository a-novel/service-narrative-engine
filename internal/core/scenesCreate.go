package core

import (
	"context"
	"errors"
	"fmt"
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
	// BeatID is the beat the scene joins.
	BeatID uuid.UUID `validate:"required"`

	// Position is the scene's place in its beat, from 0: before the beat's scene at that position, or last
	// past the end. New scenes at one position keep the order of the request.
	Position int `validate:"min=0"`

	// Title names the scene in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Description refines the beat's description for the scene. Empty keeps the beat's.
	Description string `validate:"max=20000"`
	// Prompt refines the beat's prompt for the scene. Empty keeps the beat's.
	Prompt string `validate:"max=20000"`
}

// ScenesCreateRequest carries new scenes, each with its beat and its place in it.
type ScenesCreateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Scenes lists the scenes to add, at most 100, in any beats of the project. A beat holds at most 10.
	Scenes []ScenesCreateItem `validate:"required,min=1,max=100,dive"`
}

// ScenesCreate adds scenes to beats of a project, each at its position in its beat, and returns them in
// the order of the request. The scenes after them in their beats move down to make room. A beat the project
// does not hold answers [ErrBeatNotFound], and no scene is added.
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
			Scenes: lo.Map(request.Scenes, func(item ScenesCreateItem, index int) dao.ScenesInsertItem {
				return dao.ScenesInsertItem{
					ID: ids[index], BeatID: item.BeatID, Position: item.Position, Title: item.Title,
					Description: item.Description, Prompt: item.Prompt,
				}
			}),
			Now: time.Now(),
		})
		switch {
		case errors.Is(err, dao.ErrScenesInsertFull):
			err = errors.Join(err, ErrProjectFull)
		case errors.Is(err, dao.ErrScenesInsertBeatNotFound):
			err = errors.Join(err, ErrBeatNotFound)
		}

		if err != nil {
			return fmt.Errorf("insert scenes: %w", err)
		}

		byID := lo.KeyBy(inserted, func(entity *dao.Scene) uuid.UUID { return entity.ID })
		scenes = lo.Map(ids, func(id uuid.UUID, _ int) *Scene { return newScene(byID[id]) })

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scenes, nil
}
