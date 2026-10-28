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

// CellListDaoOwner checks that the user owns the project.
type CellListDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// CellListDao reads a page of the project's cells.
type CellListDao interface {
	Exec(ctx context.Context, request *dao.CellListRequest) ([]*dao.CellListRow, error)
}

// CellListLimitDefault is the page size of a [CellListRequest] whose Limit is zero.
const CellListLimitDefault = 100

// CellListRequest selects a page of a project's cells.
type CellListRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Limit caps the page at 500 cells. It defaults to [CellListLimitDefault] when zero.
	Limit int `validate:"min=0,max=500"`
	// Offset is how many cells come before the page, by the position of their scene, then of their layer.
	Offset int `validate:"min=0"`
}

// CellList reads a project's cells page by page, by the position of their scene, then of their
// layer. Each page carries the total, so a client knows when it holds every cell.
type CellList struct {
	daoOwner CellListDaoOwner
	dao      CellListDao
}

func NewCellList(daoOwner CellListDaoOwner, dao CellListDao) *CellList {
	return &CellList{daoOwner: daoOwner, dao: dao}
}

func (service *CellList) Exec(ctx context.Context, request *CellListRequest) (*CellPage, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.CellList")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("cells.limit", request.Limit),
		attribute.Int("cells.offset", request.Offset),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	if request.Limit == 0 {
		request.Limit = CellListLimitDefault
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	rows, err := service.dao.Exec(ctx, &dao.CellListRequest{
		ProjectID: request.ProjectID,
		Limit:     request.Limit,
		Offset:    request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list cells: %w", err))
	}

	page := &CellPage{
		Cells: lo.Map(rows, func(row *dao.CellListRow, _ int) *Cell { return newCell(&row.Cell) }),
	}

	// Every row carries the total, and a page past the end carries none.
	if len(rows) > 0 {
		page.Total = rows[0].Total
	}

	return page, nil
}
