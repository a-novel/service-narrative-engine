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

// SceneListDaoProject reads the project, to check its owner.
type SceneListDaoProject interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

// SceneListDao reads a page of the project's scenes.
type SceneListDao interface {
	Exec(ctx context.Context, request *dao.SceneListRequest) ([]*dao.SceneListRow, error)
}

// SceneListDefaultSize is the page size of a [SceneListRequest] whose Limit is zero or negative.
const SceneListDefaultSize = 100

// SceneListRequest selects a page of a project's scenes.
type SceneListRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Limit caps the page at 500 scenes. It defaults to [SceneListDefaultSize] when zero or negative.
	Limit int `validate:"max=500"`
	// Offset is how many scenes come before the page, in narrative order.
	Offset int `validate:"min=0"`
}

// SceneList reads a project's scenes page by page, in narrative order. Each page carries the total,
// so a client knows when it holds every scene.
type SceneList struct {
	daoProject SceneListDaoProject
	dao        SceneListDao
}

func NewSceneList(daoProject SceneListDaoProject, dao SceneListDao) *SceneList {
	return &SceneList{daoProject: daoProject, dao: dao}
}

func (service *SceneList) Exec(ctx context.Context, request *SceneListRequest) (*ScenePage, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneList")
	defer span.End()

	limit := request.Limit
	if limit <= 0 {
		limit = SceneListDefaultSize
	}

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("scenes.limit", limit),
		attribute.Int("scenes.offset", request.Offset),
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

	rows, err := service.dao.Exec(ctx, &dao.SceneListRequest{
		ProjectID: request.ProjectID,
		Limit:     limit,
		Offset:    request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list scenes: %w", err))
	}

	page := &ScenePage{
		Scenes: lo.Map(rows, func(row *dao.SceneListRow, _ int) *Scene { return newScene(&row.Scene, 0) }),
	}

	// Every row carries the total, and a page past the end carries none.
	if len(rows) > 0 {
		page.Total = rows[0].Total
	}

	return page, nil
}
