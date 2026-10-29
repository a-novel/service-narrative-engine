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

// SceneCreateDaoLock locks the project its owner names, so changes to its order run one at a time.
type SceneCreateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// SceneCreateDaoInsert inserts the scene, and moves the beat's scenes at or after its position down.
type SceneCreateDaoInsert interface {
	Exec(ctx context.Context, request *dao.SceneInsertRequest) (*dao.Scene, error)
}

// SceneCreateRequest carries a new scene and its place in a beat.
type SceneCreateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// BeatID is the beat the scene joins.
	BeatID uuid.UUID `validate:"required"`

	// Position is the scene's place in its beat, from 0. A position past the beat's last scene appends it.
	Position int `validate:"min=0"`

	// Title names the scene in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Description refines the beat's description for the scene. Empty keeps the beat's.
	Description string `validate:"max=20000"`
	// Prompt refines the beat's prompt for the scene. Empty keeps the beat's.
	Prompt string `validate:"max=20000"`
}

// SceneCreate adds one scene to a beat at the position the caller names. The beat's scenes from that
// position on move down by one. [ScenesCreate] adds several at once.
type SceneCreate struct {
	daoLock    SceneCreateDaoLock
	daoInsert  SceneCreateDaoInsert
	transactor transaction.Transactor
}

func NewSceneCreate(
	daoLock SceneCreateDaoLock,
	daoInsert SceneCreateDaoInsert,
	transactor transaction.Transactor,
) *SceneCreate {
	return &SceneCreate{daoLock: daoLock, daoInsert: daoInsert, transactor: transactor}
}

func (service *SceneCreate) Exec(ctx context.Context, request *SceneCreateRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneCreate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("beat.id", request.BeatID.String()),
		attribute.Int("scene.position", request.Position),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	ids, err := newIDs(ctx, 1)
	if err != nil {
		return nil, otel.ReportError(span, err)
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

		inserted, err := service.daoInsert.Exec(ctx, &dao.SceneInsertRequest{
			ID:          ids[0],
			ProjectID:   request.ProjectID,
			BeatID:      request.BeatID,
			Position:    request.Position,
			Title:       request.Title,
			Description: request.Description,
			Prompt:      request.Prompt,
			Now:         time.Now(),
		})
		switch {
		case errors.Is(err, dao.ErrSceneInsertFull):
			err = errors.Join(err, ErrProjectFull)
		case errors.Is(err, dao.ErrSceneInsertBeatNotFound):
			err = errors.Join(err, ErrBeatNotFound)
		}

		if err != nil {
			return fmt.Errorf("insert scene: %w", err)
		}

		scene = newScene(inserted)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scene, nil
}
