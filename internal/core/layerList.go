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

// LayerListDaoProject reads the project, to check its owner.
type LayerListDaoProject interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

// LayerListDao reads a page of the project's layers.
type LayerListDao interface {
	Exec(ctx context.Context, request *dao.LayerListRequest) ([]*dao.LayerListRow, error)
}

// LayerListDefaultSize is the page size of a [LayerListRequest] whose Limit is zero or negative.
const LayerListDefaultSize = 100

// LayerListRequest selects a page of a project's layers.
type LayerListRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Limit caps the page at 500 layers. It defaults to [LayerListDefaultSize] when zero or negative.
	Limit int `validate:"max=500"`
	// Offset is how many layers come before the page, in order.
	Offset int `validate:"min=0"`
}

// LayerList reads a project's layers page by page, in order. Each page carries the total, so a
// client knows when it holds every layer.
type LayerList struct {
	daoProject LayerListDaoProject
	dao        LayerListDao
}

func NewLayerList(daoProject LayerListDaoProject, dao LayerListDao) *LayerList {
	return &LayerList{daoProject: daoProject, dao: dao}
}

func (service *LayerList) Exec(ctx context.Context, request *LayerListRequest) (*LayerPage, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.LayerList")
	defer span.End()

	limit := request.Limit
	if limit <= 0 {
		limit = LayerListDefaultSize
	}

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("layers.limit", limit),
		attribute.Int("layers.offset", request.Offset),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	project, err := service.daoProject.Exec(ctx, &dao.ProjectSelectRequest{ID: request.ProjectID})
	if errors.Is(err, dao.ErrProjectSelectNotFound) {
		err = errors.Join(err, ErrProjectNotFound)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select project: %w", err))
	}

	err = checkOwner(project, request.OwnerID, ErrProjectNotFound)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	rows, err := service.dao.Exec(ctx, &dao.LayerListRequest{
		ProjectID: request.ProjectID,
		Limit:     limit,
		Offset:    request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list layers: %w", err))
	}

	page := &LayerPage{
		Layers: lo.Map(rows, func(row *dao.LayerListRow, _ int) *Layer { return newLayer(&row.Layer, 0) }),
	}

	// Every row carries the total, and a page past the end carries none.
	if len(rows) > 0 {
		page.Total = rows[0].Total
	}

	return page, nil
}
