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

// LayerListDaoOwner checks that the user owns the project.
type LayerListDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// LayerListDao reads a page of the project's layers.
type LayerListDao interface {
	Exec(ctx context.Context, request *dao.LayerListRequest) ([]*dao.LayerListRow, error)
}

// LayerListLimitDefault is the page size of a [LayerListRequest] whose Limit is zero.
const LayerListLimitDefault = 100

// LayerListRequest selects a page of a project's layers.
type LayerListRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Limit caps the page at 500 layers. It defaults to [LayerListLimitDefault] when zero.
	Limit int `validate:"min=0,max=500"`
	// Offset is how many layers come before the page, in order.
	Offset int `validate:"min=0"`
}

// LayerList reads a project's layers page by page, in order. Each page carries the total, so a
// client knows when it holds every layer.
type LayerList struct {
	daoOwner LayerListDaoOwner
	dao      LayerListDao
}

func NewLayerList(daoOwner LayerListDaoOwner, dao LayerListDao) *LayerList {
	return &LayerList{daoOwner: daoOwner, dao: dao}
}

func (service *LayerList) Exec(ctx context.Context, request *LayerListRequest) (*LayerPage, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerList")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("layers.limit", request.Limit),
		attribute.Int("layers.offset", request.Offset),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	if request.Limit == 0 {
		request.Limit = LayerListLimitDefault
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	rows, err := service.dao.Exec(ctx, &dao.LayerListRequest{
		ProjectID: request.ProjectID,
		Limit:     request.Limit,
		Offset:    request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list layers: %w", err))
	}

	page := &LayerPage{
		Layers: lo.Map(rows, func(row *dao.LayerListRow, _ int) *Layer { return newLayer(&row.Layer) }),
	}

	// Every row carries the total, and a page past the end carries none.
	if len(rows) > 0 {
		page.Total = rows[0].Total
	}

	return page, nil
}
