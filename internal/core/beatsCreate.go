package core

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// BeatsCreateDaoLock locks the project its owner names, so changes to its order run one at a time.
type BeatsCreateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// BeatsCreateDaoInsert inserts the beats, and moves the beats at or after their position down.
type BeatsCreateDaoInsert interface {
	Exec(ctx context.Context, request *dao.BeatsInsertRequest) ([]*dao.Beat, error)
}

// BeatsCreateItem is one beat of a [BeatsCreateRequest].
type BeatsCreateItem struct {
	// Title names the beat in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Color is the beat's color in the composer, as a #rrggbb hex code in either case.
	Color string `validate:"required,len=7,hexcolor"`
	// Description says what the beat is for, written for the author.
	Description string `validate:"max=20000"`
	// Prompt holds extra instructions for the model when it writes about the beat.
	Prompt string `validate:"max=20000"`
}

// BeatsCreateRequest carries new beats and their place in the project.
type BeatsCreateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Position is the place of the first new beat in narrative order, from 0, and the others follow it.
	// A position past the last beat appends them.
	Position int `validate:"min=0"`
	// Beats lists the beats to add, in narrative order, at most 100.
	Beats []BeatsCreateItem `validate:"required,min=1,max=100,dive"`
}

// BeatsCreate adds beats to a project at the position the caller names. The beats from that
// position on move down to make room.
type BeatsCreate struct {
	daoLock    BeatsCreateDaoLock
	daoInsert  BeatsCreateDaoInsert
	transactor transaction.Transactor
}

func NewBeatsCreate(
	daoLock BeatsCreateDaoLock,
	daoInsert BeatsCreateDaoInsert,
	transactor transaction.Transactor,
) *BeatsCreate {
	return &BeatsCreate{daoLock: daoLock, daoInsert: daoInsert, transactor: transactor}
}

func (service *BeatsCreate) Exec(ctx context.Context, request *BeatsCreateRequest) ([]*Beat, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.BeatsCreate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("beats.position", request.Position),
		attribute.Int("beats.count", len(request.Beats)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	ids, err := newIDs(ctx, len(request.Beats))
	if err != nil {
		return nil, otel.ReportError(span, err)
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

		inserted, err := service.daoInsert.Exec(ctx, &dao.BeatsInsertRequest{
			ProjectID: request.ProjectID,
			Position:  request.Position,
			Beats: lo.Map(request.Beats, func(item BeatsCreateItem, index int) dao.BeatsInsertItem {
				return dao.BeatsInsertItem{
					ID: ids[index], Title: item.Title, Color: strings.ToLower(item.Color),
					Description: item.Description, Prompt: item.Prompt,
				}
			}),
			Now: time.Now(),
		})
		if errors.Is(err, dao.ErrBeatsInsertFull) {
			err = errors.Join(err, ErrProjectFull)
		}

		if err != nil {
			return fmt.Errorf("insert beats: %w", err)
		}

		slices.SortFunc(inserted, func(a, b *dao.Beat) int { return cmp.Compare(a.Position, b.Position) })
		beats = lo.Map(inserted, func(entity *dao.Beat, _ int) *Beat { return newBeat(entity) })

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return beats, nil
}
