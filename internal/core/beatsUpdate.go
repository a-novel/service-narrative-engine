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

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// BeatsUpdateDaoOwner checks that the user owns the project.
type BeatsUpdateDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// BeatsUpdateDao writes the beats' new fields, all of them or none.
type BeatsUpdateDao interface {
	Exec(ctx context.Context, request *dao.BeatsUpdateRequest) ([]*dao.Beat, error)
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
// order of the request. [BeatUpdate] updates one.
type BeatsUpdate struct {
	daoOwner BeatsUpdateDaoOwner
	dao      BeatsUpdateDao
}

func NewBeatsUpdate(daoOwner BeatsUpdateDaoOwner, dao BeatsUpdateDao) *BeatsUpdate {
	return &BeatsUpdate{daoOwner: daoOwner, dao: dao}
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

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	updated, err := service.dao.Exec(ctx, &dao.BeatsUpdateRequest{
		ProjectID: request.ProjectID,
		Beats: lo.Map(request.Beats, func(item BeatsUpdateItem, _ int) dao.BeatsUpdateItem {
			return dao.BeatsUpdateItem{
				ID: item.ID, Title: item.Title, Color: strings.ToLower(item.Color),
				Description: item.Description, Prompt: item.Prompt,
			}
		}),
		Now: time.Now(),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("update beats: %w", err))
	}

	// The update writes none of the beats when the project does not hold one of them.
	if len(updated) != len(request.Beats) {
		return nil, otel.ReportError(span, fmt.Errorf(
			"%w: %d of %d beats", ErrBeatNotFound, len(request.Beats)-len(updated), len(request.Beats),
		))
	}

	byID := lo.KeyBy(updated, func(entity *dao.Beat) uuid.UUID { return entity.ID })

	return lo.Map(request.Beats, func(item BeatsUpdateItem, _ int) *Beat { return newBeat(byID[item.ID]) }), nil
}
