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

// BeatMoveDaoLock locks the project its owner names, so changes to its order run one at a time.
type BeatMoveDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// BeatMoveDao moves the beat, and shifts the beats between its old and new position.
type BeatMoveDao interface {
	Exec(ctx context.Context, request *dao.BeatMoveRequest) (*dao.Beat, error)
}

// BeatMoveRequest names the beat to move and its new position.
type BeatMoveRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`

	// Position is the beat's place in narrative order after the move, from 0. A position past the last
	// beat moves it last.
	Position int `validate:"min=0"`
}

// BeatMove moves a beat within its project, and returns it at its new position. The beats between its
// old and new position shift by one.
type BeatMove struct {
	daoLock    BeatMoveDaoLock
	dao        BeatMoveDao
	transactor transaction.Transactor
}

func NewBeatMove(
	daoLock BeatMoveDaoLock,
	dao BeatMoveDao,
	transactor transaction.Transactor,
) *BeatMove {
	return &BeatMove{daoLock: daoLock, dao: dao, transactor: transactor}
}

func (service *BeatMove) Exec(ctx context.Context, request *BeatMoveRequest) (*Beat, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.BeatMove")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("beat.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("beat.position", request.Position),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	var beat *Beat

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		moved, err := service.dao.Exec(ctx, &dao.BeatMoveRequest{
			ProjectID: request.ProjectID, ID: request.ID, Position: request.Position,
		})
		if errors.Is(err, dao.ErrBeatMoveNotFound) {
			err = errors.Join(err, ErrBeatNotFound)
		}

		if err != nil {
			return fmt.Errorf("move beat: %w", err)
		}

		beat = newBeat(moved)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return beat, nil
}
