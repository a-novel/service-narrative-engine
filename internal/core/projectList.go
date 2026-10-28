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

// ProjectListDao reads a page of the caller's projects.
type ProjectListDao interface {
	Exec(ctx context.Context, request *dao.ProjectListRequest) ([]*dao.ProjectListRow, error)
}

// ProjectListLimitDefault is the page size of a [ProjectListRequest] whose Limit is zero.
const ProjectListLimitDefault = 20

// ProjectListRequest selects a page of the caller's projects.
type ProjectListRequest struct {
	OwnerID uuid.UUID `validate:"required"`

	// Limit caps the page at 100 projects. It defaults to [ProjectListLimitDefault] when zero.
	Limit int `validate:"min=0,max=100"`
	// Offset is how many projects come before the page, newest first.
	Offset int `validate:"min=0"`
}

// ProjectList reads the caller's projects page by page, newest first. Each page carries the total.
type ProjectList struct {
	dao ProjectListDao
}

func NewProjectList(dao ProjectListDao) *ProjectList {
	return &ProjectList{dao: dao}
}

func (service *ProjectList) Exec(ctx context.Context, request *ProjectListRequest) (*ProjectPage, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectList")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("projects.limit", request.Limit),
		attribute.Int("projects.offset", request.Offset),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	if request.Limit == 0 {
		request.Limit = ProjectListLimitDefault
	}

	rows, err := service.dao.Exec(ctx, &dao.ProjectListRequest{
		OwnerID: request.OwnerID,
		Limit:   request.Limit,
		Offset:  request.Offset,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list projects: %w", err))
	}

	page := &ProjectPage{
		Projects: lo.Map(rows, func(row *dao.ProjectListRow, _ int) *Project { return newProject(&row.Project) }),
	}

	// Every row carries the total, and a page past the end carries none.
	if len(rows) > 0 {
		page.Total = rows[0].Total
	}

	return page, nil
}
