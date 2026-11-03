package core

import (
	"context"
	"encoding/json"
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

// BeatRecapWriteDaoLock locks the project its owner names, so the writes to its bytes run one at a time.
type BeatRecapWriteDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// BeatRecapWriteDao writes the recap of a beat of the project.
type BeatRecapWriteDao interface {
	Exec(ctx context.Context, request *dao.BeatRecapUpdateRequest) error
}

// BeatRecapWriteDaoBytes reads how many bytes the project stores, the recap included.
type BeatRecapWriteDaoBytes interface {
	Exec(ctx context.Context, request *dao.ProjectBytesRequest) (*dao.ProjectBytes, error)
}

// BeatRecapWriteRequest carries the recap a consolidation wrote for a beat.
type BeatRecapWriteRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	BeatID    uuid.UUID `validate:"required"`

	// Recap is the output of the consolidation, a JSON document that follows the consolidate output schema.
	Recap json.RawMessage `validate:"required,max=2097152"`
	// BeatUpdatedAt is the beat's UpdatedAt the consolidation read, as its context reports it.
	BeatUpdatedAt time.Time `validate:"required"`
	// Sources lists, for each scene of the beat in order, the manuscript cell the consolidation read, as its
	// context reports them. A beat holds at most 10 scenes.
	Sources []*BeatRecapSource `validate:"max=10,unique=SceneID,dive,required"`
}

// BeatRecapWrite stores the recap a consolidation wrote for a beat of the user's project, stamped with the
// versions of the beat and of its scenes' manuscript cells it read, and leaves the beat's UpdatedAt as it
// is. It refuses a recap that read the beat at a version older than the stored recap's, or newer than the
// beat's own, and one that would bring the project past [ProjectBytesMax] bytes.
type BeatRecapWrite struct {
	daoLock    BeatRecapWriteDaoLock
	dao        BeatRecapWriteDao
	daoBytes   BeatRecapWriteDaoBytes
	transactor transaction.Transactor
}

func NewBeatRecapWrite(
	daoLock BeatRecapWriteDaoLock,
	dao BeatRecapWriteDao,
	daoBytes BeatRecapWriteDaoBytes,
	transactor transaction.Transactor,
) *BeatRecapWrite {
	return &BeatRecapWrite{daoLock: daoLock, dao: dao, daoBytes: daoBytes, transactor: transactor}
}

func (service *BeatRecapWrite) Exec(ctx context.Context, request *BeatRecapWriteRequest) error {
	ctx, span := otel.Tracer().Start(ctx, "core.BeatRecapWrite")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("beat.id", request.BeatID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validateRecap(request, request.Recap)
	if err != nil {
		return otel.ReportError(span, err)
	}

	sources, err := json.Marshal(lo.Map(request.Sources, func(source *BeatRecapSource, _ int) *dao.BeatRecapSource {
		return &dao.BeatRecapSource{SceneID: source.SceneID, CellUpdatedAt: source.CellUpdatedAt}
	}))
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("marshal sources: %w", err))
	}

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		err = service.dao.Exec(ctx, &dao.BeatRecapUpdateRequest{
			ProjectID:     request.ProjectID,
			ID:            request.BeatID,
			Recap:         request.Recap,
			BeatUpdatedAt: request.BeatUpdatedAt,
			Sources:       sources,
		})

		switch {
		case errors.Is(err, dao.ErrBeatRecapUpdateNotFound):
			err = errors.Join(err, ErrBeatNotFound)
		case errors.Is(err, dao.ErrBeatRecapUpdateConflict):
			err = errors.Join(err, ErrRecapConflict)
		}

		if err != nil {
			return fmt.Errorf("write recap: %w", err)
		}

		err = keepProjectBudget(ctx, service.daoBytes, projectBudgetOpts{projectID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("keep project budget: %w", err)
		}

		return nil
	})
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return nil
}
