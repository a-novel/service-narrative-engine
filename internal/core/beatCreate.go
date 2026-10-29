package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// BeatCreateDaoLock locks the project its owner names, so changes to its order run one at a time.
type BeatCreateDaoLock interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// BeatCreateDaoInsert inserts the beat, and moves the beats at or after its position down.
type BeatCreateDaoInsert interface {
	Exec(ctx context.Context, request *dao.BeatInsertRequest) (*dao.Beat, error)
}

// BeatCreateRequest carries a new beat and its place in the project.
type BeatCreateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Position is the beat's place in narrative order, from 0. A position past the last beat appends it.
	Position int `validate:"min=0"`

	// Title names the beat in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Color is the beat's color in the composer, as a #rrggbb hex code in either case.
	Color string `validate:"required,len=7,hexcolor"`
	// Description says what the beat is for, written for the author.
	Description string `validate:"max=20000"`
	// Prompt holds extra instructions for the model when it writes about the beat.
	Prompt string `validate:"max=20000"`
}

// BeatCreate adds one beat to a project at the position the caller names. The beats from that
// position on move down by one. [BeatsCreate] adds several at once.
type BeatCreate struct {
	daoLock    BeatCreateDaoLock
	daoInsert  BeatCreateDaoInsert
	transactor transaction.Transactor
}

func NewBeatCreate(
	daoLock BeatCreateDaoLock,
	daoInsert BeatCreateDaoInsert,
	transactor transaction.Transactor,
) *BeatCreate {
	return &BeatCreate{daoLock: daoLock, daoInsert: daoInsert, transactor: transactor}
}

func (service *BeatCreate) Exec(ctx context.Context, request *BeatCreateRequest) (*Beat, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.BeatCreate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("beat.position", request.Position),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	ids, err := newIDs(ctx, 1)
	if err != nil {
		return nil, otel.ReportError(span, err)
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

		inserted, err := service.daoInsert.Exec(ctx, &dao.BeatInsertRequest{
			ID:          ids[0],
			ProjectID:   request.ProjectID,
			Position:    request.Position,
			Title:       request.Title,
			Color:       strings.ToLower(request.Color),
			Description: request.Description,
			Prompt:      request.Prompt,
			Now:         time.Now(),
		})
		if errors.Is(err, dao.ErrBeatInsertFull) {
			err = errors.Join(err, ErrProjectFull)
		}

		if err != nil {
			return fmt.Errorf("insert beat: %w", err)
		}

		beat = newBeat(inserted)

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return beat, nil
}
