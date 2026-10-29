package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// SceneMoveDaoLock locks the project its owner names, so changes to its order run one at a time.
type SceneMoveDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// SceneMoveDao moves the scene, and closes and opens the gaps in its old and new beat.
type SceneMoveDao interface {
	Exec(ctx context.Context, request *dao.SceneMoveRequest) (*dao.Scene, error)
}

// SceneMoveRequest names the scene to move, and the beat and position it moves to.
type SceneMoveRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`

	// BeatID is the beat that holds the scene after the move: its own, or another beat of the project.
	BeatID uuid.UUID `validate:"required"`
	// Position is the scene's place in that beat after the move, from 0. A position past the beat's last
	// scene moves it last.
	Position int `validate:"min=0"`
}

// SceneMove moves a scene to a position in a beat of its project, and returns it there. Its old beat closes
// the gap and the target beat opens one, so the scene keeps its ID and its cells.
type SceneMove struct {
	daoLock    SceneMoveDaoLock
	dao        SceneMoveDao
	transactor transaction.Transactor
}

func NewSceneMove(
	daoLock SceneMoveDaoLock,
	dao SceneMoveDao,
	transactor transaction.Transactor,
) *SceneMove {
	return &SceneMove{daoLock: daoLock, dao: dao, transactor: transactor}
}

func (service *SceneMove) Exec(ctx context.Context, request *SceneMoveRequest) (*Scene, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneMove")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("scene.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("beat.id", request.BeatID.String()),
		attribute.Int("scene.position", request.Position),
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

		moved, err := service.dao.Exec(ctx, &dao.SceneMoveRequest{
			ProjectID: request.ProjectID, ID: request.ID, BeatID: request.BeatID, Position: request.Position,
		})

		switch {
		case errors.Is(err, dao.ErrSceneMoveNotFound):
			err = errors.Join(err, ErrSceneNotFound)
		case errors.Is(err, dao.ErrSceneMoveBeatNotFound):
			err = errors.Join(err, ErrBeatNotFound)
		case errors.Is(err, dao.ErrSceneMoveFull):
			err = errors.Join(err, ErrProjectFull)
		}

		if err != nil {
			return fmt.Errorf("move scene: %w", err)
		}

		scene = newScene(moved)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return scene, nil
}
