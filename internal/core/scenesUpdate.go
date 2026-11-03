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

// ScenesUpdateDaoLock locks the project its owner names, so the writes to its bytes run one at a time.
type ScenesUpdateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// ScenesUpdateDao writes the scenes' new fields, all of them or none.
type ScenesUpdateDao interface {
	Exec(ctx context.Context, request *dao.ScenesUpdateRequest) ([]*dao.Scene, error)
}

// ScenesUpdateDaoBytes reads how many bytes the project stores, the write included.
type ScenesUpdateDaoBytes interface {
	Exec(ctx context.Context, request *dao.ProjectBytesRequest) (*dao.ProjectBytes, error)
}

// ScenesUpdateItem is one scene of a [ScenesUpdateRequest].
type ScenesUpdateItem struct {
	ID uuid.UUID `validate:"required"`

	// Title names the scene in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Description refines the beat's description for the scene. Empty keeps the beat's.
	Description string `validate:"max=20000"`
	// Prompt refines the beat's prompt for the scene. Empty keeps the beat's.
	Prompt string `validate:"max=20000"`
}

// ScenesUpdateRequest carries the new fields of scenes.
type ScenesUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// Scenes lists the scenes to update, each once, at most 100.
	Scenes []ScenesUpdateItem `validate:"required,min=1,max=100,unique=ID,dive"`
}

// ScenesUpdate replaces the fields of several scenes at once, all of them or none, and returns them in the
// order of the request. [SceneUpdate] updates one. It writes nothing when the project would store more than
// [ProjectBytesMax] bytes.
type ScenesUpdate struct {
	daoLock    ScenesUpdateDaoLock
	dao        ScenesUpdateDao
	daoBytes   ScenesUpdateDaoBytes
	transactor transaction.Transactor
}

func NewScenesUpdate(
	daoLock ScenesUpdateDaoLock,
	dao ScenesUpdateDao,
	daoBytes ScenesUpdateDaoBytes,
	transactor transaction.Transactor,
) *ScenesUpdate {
	return &ScenesUpdate{daoLock: daoLock, dao: dao, daoBytes: daoBytes, transactor: transactor}
}

func (service *ScenesUpdate) Exec(ctx context.Context, request *ScenesUpdateRequest) ([]*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ScenesUpdate")
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

	updateItems := make([]dao.ScenesUpdateItem, len(request.Scenes))

	for i, item := range request.Scenes {
		updateItems[i] = dao.ScenesUpdateItem{
			ID:          item.ID,
			Title:       item.Title,
			Description: item.Description,
			Prompt:      item.Prompt,
		}
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

		updated, err := service.dao.Exec(ctx, &dao.ScenesUpdateRequest{
			ProjectID: request.ProjectID,
			Scenes:    updateItems,
			Now:       time.Now(),
		})
		if err != nil {
			return fmt.Errorf("update scenes: %w", err)
		}

		// The update writes none of the scenes when the project does not hold one of them.
		if len(updated) != len(request.Scenes) {
			return fmt.Errorf(
				"%w: %d of %d scenes", ErrSceneNotFound, len(request.Scenes)-len(updated), len(request.Scenes),
			)
		}

		err = keepProjectBudget(ctx, service.daoBytes, projectBudgetOpts{projectID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("keep project budget: %w", err)
		}

		byID := lo.KeyBy(updated, func(entity *dao.Scene) uuid.UUID { return entity.ID })
		scenes = lo.Map(request.Scenes, func(item ScenesUpdateItem, _ int) *Scene { return newScene(byID[item.ID]) })

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scenes, nil
}
