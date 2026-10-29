package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// BeatListDaoOwner checks that the user owns the project.
type BeatListDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// BeatListDao reads a page of the project's beats.
type BeatListDao interface {
	Exec(ctx context.Context, request *dao.BeatListRequest) ([]*dao.BeatListRow, error)
}

// BeatListLimitDefault is the page size of a [BeatListRequest] whose Limit is zero.
const BeatListLimitDefault = 100

// BeatListRequest selects a page of a project's beats.
type BeatListRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Limit caps the page at 500 beats. It defaults to [BeatListLimitDefault] when zero.
	Limit int `validate:"min=0,max=500"`
	// Offset is how many beats come before the page, in narrative order.
	Offset int `validate:"min=0"`
}

// BeatList reads a project's beats page by page, in narrative order. Each page carries the total,
// so a client knows when it holds every beat.
type BeatList struct {
	daoOwner BeatListDaoOwner
	dao      BeatListDao
}

func NewBeatList(daoOwner BeatListDaoOwner, dao BeatListDao) *BeatList {
	return &BeatList{daoOwner: daoOwner, dao: dao}
}

func (service *BeatList) Exec(ctx context.Context, request *BeatListRequest) (*BeatPage, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.BeatList")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("beats.limit", request.Limit),
		attribute.Int("beats.offset", request.Offset),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	if request.Limit == 0 {
		request.Limit = BeatListLimitDefault
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	rows, err := service.dao.Exec(ctx, &dao.BeatListRequest{
		ProjectID: request.ProjectID,
		Limit:     request.Limit,
		Offset:    request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list beats: %w", err))
	}

	page := &BeatPage{
		Beats: lo.Map(rows, func(row *dao.BeatListRow, _ int) *Beat { return newBeat(&row.Beat) }),
	}

	// Every row carries the total, and a page past the end carries none.
	if len(rows) > 0 {
		page.Total = rows[0].Total
	}

	return page, nil
}
