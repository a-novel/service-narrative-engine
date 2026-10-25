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
	Exec(ctx context.Context, request *dao.ProjectListRequest) ([]*dao.Project, error)
}

// ProjectListDefaultSize is the page size of a [ProjectListRequest] whose Limit is zero or negative.
const ProjectListDefaultSize = 20

// ProjectListRequest selects a page of the caller's projects.
type ProjectListRequest struct {
	OwnerID uuid.UUID `validate:"required"`
	// Limit caps the page at 100 projects. It defaults to [ProjectListDefaultSize] when zero or
	// negative.
	Limit int `validate:"max=100"`
	// After is the ID of the last project of the previous page. The page starts with the next older
	// project, or with the newest one when After is uuid.Nil.
	After uuid.UUID
}

// ProjectList reads the caller's projects, newest first.
type ProjectList struct {
	dao ProjectListDao
}

func NewProjectList(dao ProjectListDao) *ProjectList {
	return &ProjectList{dao: dao}
}

func (service *ProjectList) Exec(ctx context.Context, request *ProjectListRequest) ([]*Project, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectList")
	defer span.End()

	limit := request.Limit
	if limit <= 0 {
		limit = ProjectListDefaultSize
	}

	// Project IDs are UUIDv7, so uuid.Max sorts after every project.
	after := request.After
	if after == uuid.Nil {
		after = uuid.Max
	}

	span.SetAttributes(
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("projects.limit", limit),
		attribute.String("projects.after", request.After.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	entities, err := service.dao.Exec(ctx, &dao.ProjectListRequest{OwnerID: request.OwnerID, Limit: limit, After: after})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list projects: %w", err))
	}

	return lo.Map(entities, func(entity *dao.Project, _ int) *Project { return newProject(entity) }), nil
}
