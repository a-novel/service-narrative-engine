package core

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// BeatsDeleteDaoLock locks the project its owner names, so changes to its order run one at a time.
type BeatsDeleteDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// BeatsDeleteDao removes the beats, and moves the beats after them up.
type BeatsDeleteDao interface {
	Exec(ctx context.Context, request *dao.BeatsDeleteRequest) ([]*dao.Beat, error)
}

// BeatsDeleteRequest names the beats to delete.
type BeatsDeleteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// IDs lists the beats to delete, each once, at most 100.
	IDs []uuid.UUID `validate:"required,min=1,max=100,unique,dive,required"`
}

// BeatsDelete deletes beats with their scenes and cells, and moves the beats after them up to close the
// gaps. It deletes all of them or none.
type BeatsDelete struct {
	daoLock    BeatsDeleteDaoLock
	dao        BeatsDeleteDao
	transactor transaction.Transactor
}

func NewBeatsDelete(
	daoLock BeatsDeleteDaoLock,
	dao BeatsDeleteDao,
	transactor transaction.Transactor,
) *BeatsDelete {
	return &BeatsDelete{daoLock: daoLock, dao: dao, transactor: transactor}
}

func (service *BeatsDelete) Exec(ctx context.Context, request *BeatsDeleteRequest) ([]*Beat, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.BeatsDelete")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("beats.count", len(request.IDs)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	var beats []*Beat

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		deleted, err := service.dao.Exec(ctx, &dao.BeatsDeleteRequest{ProjectID: request.ProjectID, IDs: request.IDs})
		if err != nil {
			return fmt.Errorf("delete beats: %w", err)
		}

		// A miss rolls the whole delete back.
		if len(deleted) != len(request.IDs) {
			return fmt.Errorf("%w: %d of %d beats", ErrBeatNotFound, len(request.IDs)-len(deleted), len(request.IDs))
		}

		slices.SortFunc(deleted, func(a, b *dao.Beat) int { return cmp.Compare(a.Position, b.Position) })
		beats = lo.Map(deleted, func(entity *dao.Beat, _ int) *Beat { return newBeat(entity) })

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return beats, nil
}
