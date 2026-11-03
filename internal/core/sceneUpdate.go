package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// SceneUpdateDaoLock locks the project its owner names, so the writes to its bytes run one at a time.
type SceneUpdateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// SceneUpdateDao writes the scene's new fields.
type SceneUpdateDao interface {
	Exec(ctx context.Context, request *dao.SceneUpdateRequest) (*dao.Scene, error)
}

// SceneUpdateDaoBytes reads how many bytes the project stores, the write included.
type SceneUpdateDaoBytes interface {
	Exec(ctx context.Context, request *dao.ProjectBytesRequest) (*dao.ProjectBytes, error)
}

// SceneUpdateRequest carries a scene's new fields.
type SceneUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`

	// Title names the scene in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Description refines the beat's description for the scene. Empty keeps the beat's.
	Description string `validate:"max=20000"`
	// Prompt refines the beat's prompt for the scene. Empty keeps the beat's.
	Prompt string `validate:"max=20000"`
}

// SceneUpdate replaces a scene's title, description and prompt. It writes nothing when the project would
// store more than [ProjectBytesMax] bytes.
type SceneUpdate struct {
	daoLock    SceneUpdateDaoLock
	dao        SceneUpdateDao
	daoBytes   SceneUpdateDaoBytes
	transactor transaction.Transactor
}

func NewSceneUpdate(
	daoLock SceneUpdateDaoLock,
	dao SceneUpdateDao,
	daoBytes SceneUpdateDaoBytes,
	transactor transaction.Transactor,
) *SceneUpdate {
	return &SceneUpdate{daoLock: daoLock, dao: dao, daoBytes: daoBytes, transactor: transactor}
}

func (service *SceneUpdate) Exec(ctx context.Context, request *SceneUpdateRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("scene.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	var scene *Scene

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		entity, err := service.dao.Exec(ctx, &dao.SceneUpdateRequest{
			ProjectID:   request.ProjectID,
			ID:          request.ID,
			Title:       request.Title,
			Description: request.Description,
			Prompt:      request.Prompt,
			Now:         time.Now(),
		})
		if errors.Is(err, dao.ErrSceneUpdateNotFound) {
			err = errors.Join(err, ErrSceneNotFound)
		}

		if err != nil {
			return fmt.Errorf("update scene: %w", err)
		}

		err = keepProjectBudget(ctx, service.daoBytes, projectBudgetOpts{projectID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("keep project budget: %w", err)
		}

		scene = newScene(entity)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scene, nil
}
