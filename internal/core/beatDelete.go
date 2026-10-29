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

// BeatDeleteDaoLock locks the project its owner names, so changes to its order run one at a time.
type BeatDeleteDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// BeatDeleteDao removes the beat, and moves the beats after it up.
type BeatDeleteDao interface {
	Exec(ctx context.Context, request *dao.BeatDeleteRequest) (*dao.Beat, error)
}

// BeatDeleteRequest names the beat to delete.
type BeatDeleteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`
}

// BeatDelete deletes one beat with its scenes and their cells, and moves the beats after it up by one.
// [BeatsDelete] deletes several at once.
type BeatDelete struct {
	daoLock    BeatDeleteDaoLock
	dao        BeatDeleteDao
	transactor transaction.Transactor
}

func NewBeatDelete(daoLock BeatDeleteDaoLock, dao BeatDeleteDao, transactor transaction.Transactor) *BeatDelete {
	return &BeatDelete{daoLock: daoLock, dao: dao, transactor: transactor}
}

func (service *BeatDelete) Exec(ctx context.Context, request *BeatDeleteRequest) (*Beat, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.BeatDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("beat.id", request.ID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
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

		deleted, err := service.dao.Exec(ctx, &dao.BeatDeleteRequest{ProjectID: request.ProjectID, ID: request.ID})
		if errors.Is(err, dao.ErrBeatDeleteNotFound) {
			err = errors.Join(err, ErrBeatNotFound)
		}

		if err != nil {
			return fmt.Errorf("delete beat: %w", err)
		}

		beat = newBeat(deleted)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return beat, nil
}
