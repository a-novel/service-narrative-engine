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

// SceneListDaoOwner checks that the user owns the project.
type SceneListDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// SceneListDao reads a page of the project's scenes.
type SceneListDao interface {
	Exec(ctx context.Context, request *dao.SceneListRequest) ([]*dao.SceneListRow, error)
}

// SceneListLimitDefault is the page size of a [SceneListRequest] whose Limit is zero.
const SceneListLimitDefault = 100

// SceneListRequest selects a page of a project's scenes.
type SceneListRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Limit caps the page at 500 scenes. It defaults to [SceneListLimitDefault] when zero.
	Limit int `validate:"min=0,max=500"`
	// Offset is how many scenes come before the page, in narrative order.
	Offset int `validate:"min=0"`
}

// SceneList reads a project's scenes page by page, in narrative order. Each page carries the total,
// so a client knows when it holds every scene.
type SceneList struct {
	daoOwner SceneListDaoOwner
	dao      SceneListDao
}

func NewSceneList(daoOwner SceneListDaoOwner, dao SceneListDao) *SceneList {
	return &SceneList{daoOwner: daoOwner, dao: dao}
}

func (service *SceneList) Exec(ctx context.Context, request *SceneListRequest) (*ScenePage, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.SceneList")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("scenes.limit", request.Limit),
		attribute.Int("scenes.offset", request.Offset),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	if request.Limit == 0 {
		request.Limit = SceneListLimitDefault
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	rows, err := service.dao.Exec(ctx, &dao.SceneListRequest{
		ProjectID: request.ProjectID,
		Limit:     request.Limit,
		Offset:    request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list scenes: %w", err))
	}

	page := &ScenePage{
		Scenes: lo.Map(rows, func(row *dao.SceneListRow, _ int) *Scene { return newScene(&row.Scene) }),
	}

	// Every row carries the total, and a page past the end carries none.
	if len(rows) > 0 {
		page.Total = rows[0].Total
	}

	return page, nil
}
