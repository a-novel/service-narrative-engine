package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// BeatsUpdateDaoLock locks the project its owner names, so the writes to its bytes run one at a time.
type BeatsUpdateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// BeatsUpdateDao writes the beats' new fields, all of them or none.
type BeatsUpdateDao interface {
	Exec(ctx context.Context, request *dao.BeatsUpdateRequest) ([]*dao.Beat, error)
}

// BeatsUpdateDaoBytes reads how many bytes the project stores, the write included.
type BeatsUpdateDaoBytes interface {
	Exec(ctx context.Context, request *dao.ProjectBytesRequest) (*dao.ProjectBytes, error)
}

// BeatsUpdateItem is one beat of a [BeatsUpdateRequest].
type BeatsUpdateItem struct {
	ID uuid.UUID `validate:"required"`

	// Title names the beat in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Color is the beat's color in the composer, as a #rrggbb hex code in either case.
	Color string `validate:"required,len=7,hexcolor"`
	// Description says what the beat is for, written for the author.
	Description string `validate:"max=20000"`
	// Prompt holds extra instructions for the model when it writes about the beat.
	Prompt string `validate:"max=20000"`
}

// BeatsUpdateRequest carries the new fields of beats.
type BeatsUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// Beats lists the beats to update, each once, at most 100.
	Beats []BeatsUpdateItem `validate:"required,min=1,max=100,unique=ID,dive"`
}

// BeatsUpdate replaces the fields of several beats at once, all of them or none, and returns them in the
// order of the request. [BeatUpdate] updates one. It writes nothing when the project would store more than
// [ProjectBytesMax] bytes.
type BeatsUpdate struct {
	daoLock    BeatsUpdateDaoLock
	dao        BeatsUpdateDao
	daoBytes   BeatsUpdateDaoBytes
	transactor transaction.Transactor
}

func NewBeatsUpdate(
	daoLock BeatsUpdateDaoLock,
	dao BeatsUpdateDao,
	daoBytes BeatsUpdateDaoBytes,
	transactor transaction.Transactor,
) *BeatsUpdate {
	return &BeatsUpdate{daoLock: daoLock, dao: dao, daoBytes: daoBytes, transactor: transactor}
}

func (service *BeatsUpdate) Exec(ctx context.Context, request *BeatsUpdateRequest) ([]*Beat, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.BeatsUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("beats.count", len(request.Beats)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	updateItems := make([]dao.BeatsUpdateItem, len(request.Beats))

	for i, item := range request.Beats {
		updateItems[i] = dao.BeatsUpdateItem{
			ID:          item.ID,
			Title:       item.Title,
			Color:       strings.ToLower(item.Color),
			Description: item.Description,
			Prompt:      item.Prompt,
		}
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

		updated, err := service.dao.Exec(ctx, &dao.BeatsUpdateRequest{
			ProjectID: request.ProjectID,
			Beats:     updateItems,
			Now:       time.Now(),
		})
		if err != nil {
			return fmt.Errorf("update beats: %w", err)
		}

		// The update writes none of the beats when the project does not hold one of them.
		if len(updated) != len(request.Beats) {
			return fmt.Errorf(
				"%w: %d of %d beats", ErrBeatNotFound, len(request.Beats)-len(updated), len(request.Beats),
			)
		}

		err = keepProjectBudget(ctx, service.daoBytes, projectBudgetOpts{projectID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("keep project budget: %w", err)
		}

		byID := lo.KeyBy(updated, func(entity *dao.Beat) uuid.UUID { return entity.ID })
		beats = lo.Map(request.Beats, func(item BeatsUpdateItem, _ int) *Beat { return newBeat(byID[item.ID]) })

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return beats, nil
}
