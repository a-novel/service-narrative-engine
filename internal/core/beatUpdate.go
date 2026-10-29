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

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// BeatUpdateDaoOwner checks that the user owns the project.
type BeatUpdateDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// BeatUpdateDao writes the beat's new fields.
type BeatUpdateDao interface {
	Exec(ctx context.Context, request *dao.BeatUpdateRequest) (*dao.Beat, error)
}

// BeatUpdateRequest carries a beat's new fields.
type BeatUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`

	// Title names the beat in the composer.
	Title string `validate:"required,notblank,max=200"`
	// Color is the beat's color in the composer, as a #rrggbb hex code in either case.
	Color string `validate:"required,len=7,hexcolor"`
	// Description says what the beat is for, written for the author.
	Description string `validate:"max=20000"`
	// Prompt holds extra instructions for the model when it writes about the beat.
	Prompt string `validate:"max=20000"`
}

// BeatUpdate replaces a beat's title, color, description and prompt.
type BeatUpdate struct {
	daoOwner BeatUpdateDaoOwner
	dao      BeatUpdateDao
}

func NewBeatUpdate(daoOwner BeatUpdateDaoOwner, dao BeatUpdateDao) *BeatUpdate {
	return &BeatUpdate{daoOwner: daoOwner, dao: dao}
}

func (service *BeatUpdate) Exec(ctx context.Context, request *BeatUpdateRequest) (*Beat, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.BeatUpdate")
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

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	entity, err := service.dao.Exec(ctx, &dao.BeatUpdateRequest{
		ProjectID:   request.ProjectID,
		ID:          request.ID,
		Title:       request.Title,
		Color:       strings.ToLower(request.Color),
		Description: request.Description,
		Prompt:      request.Prompt,
		Now:         time.Now(),
	})
	if errors.Is(err, dao.ErrBeatUpdateNotFound) {
		err = errors.Join(err, ErrBeatNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("update beat: %w", err))
	}

	return newBeat(entity), nil
}
