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

// CellListDaoProject reads the project, so a project of another owner reads as missing.
type CellListDaoProject interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

// CellListDao reads a page of the project's cells.
type CellListDao interface {
	Exec(ctx context.Context, request *dao.CellListRequest) ([]*dao.Cell, error)
}

// CellListDaoCount counts the project's cells.
type CellListDaoCount interface {
	Exec(ctx context.Context, request *dao.CellCountRequest) (*dao.CellCount, error)
}

// CellListDefaultSize is the page size of a [CellListRequest] whose Limit is zero or negative.
const CellListDefaultSize = 100

// CellListRequest selects a page of a project's cells.
type CellListRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Limit caps the page at 500 cells. It defaults to [CellListDefaultSize] when zero or negative.
	Limit int `validate:"max=500"`
	// AfterSceneID and AfterLayerID are the key of the last cell of the previous page. The page
	// starts with the next cell in key order, or with the first one when both are uuid.Nil.
	AfterSceneID uuid.UUID
	AfterLayerID uuid.UUID
}

// CellList reads a project's cells page by page, in the order of their keys. Each page carries the
// total, so a client that loaded the project with [ProjectGet] knows when it holds every cell.
type CellList struct {
	daoProject CellListDaoProject
	dao        CellListDao
	daoCount   CellListDaoCount
}

func NewCellList(daoProject CellListDaoProject, dao CellListDao, daoCount CellListDaoCount) *CellList {
	return &CellList{daoProject: daoProject, dao: dao, daoCount: daoCount}
}

func (service *CellList) Exec(ctx context.Context, request *CellListRequest) (*CellPage, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.CellList")
	defer span.End()

	limit := request.Limit
	if limit <= 0 {
		limit = CellListDefaultSize
	}

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("cells.limit", limit),
		attribute.String("cells.after_scene_id", request.AfterSceneID.String()),
		attribute.String("cells.after_layer_id", request.AfterLayerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	_, err = service.daoProject.Exec(ctx, &dao.ProjectSelectRequest{ID: request.ProjectID, OwnerID: request.OwnerID})
	if errors.Is(err, dao.ErrProjectSelectNotFound) {
		err = errors.Join(err, ErrProjectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select project: %w", err))
	}

	entities, err := service.dao.Exec(ctx, &dao.CellListRequest{
		ProjectID:    request.ProjectID,
		Limit:        limit,
		AfterSceneID: request.AfterSceneID,
		AfterLayerID: request.AfterLayerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list cells: %w", err))
	}

	count, err := service.daoCount.Exec(ctx, &dao.CellCountRequest{ProjectID: request.ProjectID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("count cells: %w", err))
	}

	return &CellPage{Cells: lo.Map(entities, newCell), Total: count.Count}, nil
}
